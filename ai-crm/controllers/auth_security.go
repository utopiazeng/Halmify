package controllers

import (
	"ai-crm/utils"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/redis/go-redis/v9"
)

func sha256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

const (
	sliderChallengeTTL        = 2 * time.Minute
	sliderTokenTTL            = 3 * time.Minute
	loginFailureResponseDelay = 350 * time.Millisecond
)

var (
	loginIPWindow         = 1 * time.Minute
	loginUserWindow       = 15 * time.Minute
	loginIPMaxFailures    = 30
	loginUserMaxFailures  = 5
	loginUserLockDuration = 15 * time.Minute
)

type sliderChallengeState struct {
	ID        string    `json:"id"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Verified  bool      `json:"verified"`
	Consumed  bool      `json:"consumed"`
}

type loginAttemptState struct {
	Failures  []int64 `json:"failures"`
	LockUntil int64   `json:"lock_until"`
}

type sliderVerifyRequest struct {
	ChallengeID  string  `json:"challenge_id"`
	ElapsedMS    int64   `json:"elapsed_ms"`
	Track        []int   `json:"track"`
	FinalPercent float64 `json:"final_percent"`
}

type captchaTokenPayload struct {
	ChallengeID string `json:"cid"`
	IP          string `json:"ip"`
	UAHash      string `json:"uah"`
	ExpiresAt   int64  `json:"exp"`
	Nonce       string `json:"n"`
}

type authSecurityStore struct {
	mu sync.Mutex
}

var authSecurity = &authSecurityStore{}
var loginConfigOnce sync.Once

func authRedisPrefix() string {
	prefix := strings.TrimSpace(os.Getenv("AUTH_SECURITY_REDIS_PREFIX"))
	if prefix != "" {
		return prefix
	}
	conf, _ := web.AppConfig.String("auth_security_redis_prefix")
	conf = strings.TrimSpace(conf)
	if conf == "" {
		return "auth"
	}
	return conf
}

func challengeKey(challengeID string) string {
	return fmt.Sprintf("%s:captcha:challenge:%s", authRedisPrefix(), challengeID)
}

func challengeConsumedKey(challengeID string) string {
	return fmt.Sprintf("%s:captcha:consumed:%s", authRedisPrefix(), challengeID)
}

func captchaNonceKey(nonce string) string {
	return fmt.Sprintf("%s:captcha:nonce:%s", authRedisPrefix(), strings.TrimSpace(nonce))
}

func ipStateKey(ip string) string {
	return fmt.Sprintf("%s:login:ip:%s", authRedisPrefix(), strings.TrimSpace(ip))
}

func userStateKey(username string) string {
	return fmt.Sprintf("%s:login:user:%s", authRedisPrefix(), normalizeLoginKey(username))
}

func loadLoginSecurityConfig() {
	runMode, _ := web.AppConfig.String("runmode")
	ipPerMinuteDefault := 10
	if strings.EqualFold(strings.TrimSpace(runMode), "dev") {
		ipPerMinuteDefault = 30
	}
	ipPerMinute := web.AppConfig.DefaultInt("security_login_ip_rate_per_minute", ipPerMinuteDefault)
	userFailures := web.AppConfig.DefaultInt("security_login_user_failures", 5)
	userWindowMinutes := web.AppConfig.DefaultInt("security_login_user_window_minutes", 15)
	lockMinutes := web.AppConfig.DefaultInt("security_login_lock_minutes", 15)
	if env := os.Getenv("SECURITY_LOGIN_IP_RATE_PER_MINUTE"); strings.TrimSpace(env) != "" {
		if parsed, err := strconv.Atoi(env); err == nil && parsed > 0 {
			ipPerMinute = parsed
		}
	}
	if env := os.Getenv("SECURITY_LOGIN_USER_FAILURES"); strings.TrimSpace(env) != "" {
		if parsed, err := strconv.Atoi(env); err == nil && parsed > 0 {
			userFailures = parsed
		}
	}
	if env := os.Getenv("SECURITY_LOGIN_USER_WINDOW_MINUTES"); strings.TrimSpace(env) != "" {
		if parsed, err := strconv.Atoi(env); err == nil && parsed > 0 {
			userWindowMinutes = parsed
		}
	}
	if env := os.Getenv("SECURITY_LOGIN_LOCK_MINUTES"); strings.TrimSpace(env) != "" {
		if parsed, err := strconv.Atoi(env); err == nil && parsed > 0 {
			lockMinutes = parsed
		}
	}
	if ipPerMinute <= 0 {
		ipPerMinute = ipPerMinuteDefault
	}
	if userFailures <= 0 {
		userFailures = 5
	}
	if userWindowMinutes <= 0 {
		userWindowMinutes = 15
	}
	if lockMinutes <= 0 {
		lockMinutes = 15
	}
	loginIPWindow = time.Minute
	loginIPMaxFailures = ipPerMinute
	loginUserWindow = time.Duration(userWindowMinutes) * time.Minute
	loginUserMaxFailures = userFailures
	loginUserLockDuration = time.Duration(lockMinutes) * time.Minute
}

func redisClient() *redis.Client {
	return utils.RedisClient
}

func (c *AuthController) SliderChallenge() {
	ip := c.Ctx.Input.IP()
	challengeID := utils.NewSessionID()
	now := time.Now()
	challenge := &sliderChallengeState{
		ID:        challengeID,
		IP:        ip,
		CreatedAt: now,
		ExpiresAt: now.Add(sliderChallengeTTL),
	}
	if err := saveChallenge(challenge, sliderChallengeTTL); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "验证码服务不可用")
		return
	}

	utils.Success(c.Ctx, map[string]interface{}{
		"challenge_id": challengeID,
		"expires_in":   int(sliderChallengeTTL.Seconds()),
		"track_length": 320,
		"slider_width": 48,
		"hint":         "请拖动滑块到最右侧完成验证",
	})
}

func (c *AuthController) VerifySlider() {
	var req sliderVerifyRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}

	ip := c.Ctx.Input.IP()
	now := time.Now()

	challengeID := strings.TrimSpace(req.ChallengeID)
	challenge, err := loadChallenge(challengeID)
	if err != nil || challenge == nil || challenge.IP != ip || now.After(challenge.ExpiresAt) || challenge.Consumed {
		utils.Error(c.Ctx, http.StatusBadRequest, "滑动验证码已失效，请刷新重试")
		return
	}
	if !isHumanSlide(req) {
		utils.Error(c.Ctx, http.StatusBadRequest, "滑动验证未通过，请重试")
		return
	}

	challenge.Verified = true
	if err := saveChallenge(challenge, ttlFromNow(challenge.ExpiresAt, time.Second)); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "验证码服务不可用")
		return
	}

	token, err := issueCaptchaToken(challenge, now, c.Ctx.Request.UserAgent())
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "生成验证码令牌失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"captcha_token": token,
		"expires_in":    int(sliderTokenTTL.Seconds()),
	})
}

func ttlFromNow(deadline time.Time, min time.Duration) time.Duration {
	ttl := time.Until(deadline)
	if ttl < min {
		return min
	}
	return ttl
}

func loadChallenge(challengeID string) (*sliderChallengeState, error) {
	if strings.TrimSpace(challengeID) == "" {
		return nil, nil
	}
	ctx := context.Background()
	raw, err := redisClient().Get(ctx, challengeKey(challengeID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	state := &sliderChallengeState{}
	if err := json.Unmarshal([]byte(raw), state); err != nil {
		return nil, err
	}
	return state, nil
}

func saveChallenge(challenge *sliderChallengeState, ttl time.Duration) error {
	if challenge == nil {
		return nil
	}
	ctx := context.Background()
	raw, err := json.Marshal(challenge)
	if err != nil {
		return err
	}
	return redisClient().Set(ctx, challengeKey(challenge.ID), raw, ttl).Err()
}

func (s *authSecurityStore) checkLoginAllowed(ip, username string, now time.Time) (int, string, bool) {
	ipState, err := loadLoginState(ipStateKey(ip))
	if err != nil {
		return http.StatusTooManyRequests, "登录防护繁忙，请稍后重试", false
	}
	userKey := userStateKey(username)
	userState, err := loadLoginState(userKey)
	if err != nil {
		return http.StatusTooManyRequests, "登录防护繁忙，请稍后重试", false
	}
	ipState.Failures = pruneFailureUnix(ipState.Failures, now.Add(-loginIPWindow).Unix())
	userState.Failures = pruneFailureUnix(userState.Failures, now.Add(-loginUserWindow).Unix())

	if userState.LockUntil > now.Unix() {
		return http.StatusTooManyRequests, "登录尝试过于频繁，请稍后再试", false
	}
	if len(ipState.Failures) >= loginIPMaxFailures {
		return http.StatusTooManyRequests, "登录尝试过于频繁，请稍后再试", false
	}
	_ = saveLoginState(ipStateKey(ip), ipState, loginIPWindow+loginUserLockDuration)
	_ = saveLoginState(userKey, userState, loginUserWindow+loginUserLockDuration)
	return http.StatusOK, "", true
}

func (s *authSecurityStore) markLoginFailure(ip, username string, now time.Time) {
	ipKey := ipStateKey(ip)
	userKey := userStateKey(username)

	ipState, _ := loadLoginState(ipKey)
	userState, _ := loadLoginState(userKey)

	ipState.Failures = append(pruneFailureUnix(ipState.Failures, now.Add(-loginIPWindow).Unix()), now.Unix())
	userState.Failures = append(pruneFailureUnix(userState.Failures, now.Add(-loginUserWindow).Unix()), now.Unix())
	if len(userState.Failures) >= loginUserMaxFailures {
		userState.LockUntil = now.Add(loginUserLockDuration).Unix()
	}
	_ = saveLoginState(ipKey, ipState, loginIPWindow+loginUserLockDuration)
	_ = saveLoginState(userKey, userState, loginUserWindow+loginUserLockDuration)
}

func (s *authSecurityStore) markLoginSuccess(ip, username string) {
	ctx := context.Background()
	_ = redisClient().Del(ctx, userStateKey(username)).Err()
	_ = redisClient().Del(ctx, ipStateKey(ip)).Err()
}

func loadLoginState(key string) (*loginAttemptState, error) {
	ctx := context.Background()
	raw, err := redisClient().Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return &loginAttemptState{Failures: []int64{}, LockUntil: 0}, nil
		}
		return nil, err
	}
	state := &loginAttemptState{}
	if err := json.Unmarshal([]byte(raw), state); err != nil {
		return nil, err
	}
	if state.Failures == nil {
		state.Failures = []int64{}
	}
	return state, nil
}

func saveLoginState(key string, state *loginAttemptState, ttl time.Duration) error {
	if state == nil {
		return nil
	}
	ctx := context.Background()
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return redisClient().Set(ctx, key, raw, ttl).Err()
}

func pruneFailureUnix(items []int64, cutoff int64) []int64 {
	filtered := make([]int64, 0, len(items))
	for _, item := range items {
		if item > cutoff {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func normalizeLoginKey(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func isHumanSlide(req sliderVerifyRequest) bool {
	if strings.TrimSpace(req.ChallengeID) == "" {
		return false
	}
	if req.ElapsedMS < 600 || req.ElapsedMS > 15000 {
		return false
	}
	if req.FinalPercent < 98 {
		return false
	}
	if len(req.Track) < 8 || len(req.Track) > 400 {
		return false
	}

	last := req.Track[0]
	if last < 0 || last > 100 {
		return false
	}

	backwardMoves := 0
	positiveSteps := 0
	uniqueStep := map[int]bool{}
	for i := 1; i < len(req.Track); i++ {
		current := req.Track[i]
		if current < 0 || current > 100 {
			return false
		}
		delta := current - last
		if delta > 0 {
			positiveSteps++
			uniqueStep[int(math.Abs(float64(delta)))] = true
		} else if delta < 0 {
			backwardMoves++
		}
		last = current
	}

	if last < 98 {
		return false
	}
	if positiveSteps < 5 {
		return false
	}
	if backwardMoves > 2 {
		return false
	}
	if len(uniqueStep) < 3 {
		return false
	}
	return true
}

func issueCaptchaToken(challenge *sliderChallengeState, now time.Time, userAgent string) (string, error) {
	uaHash := sha256Hex([]byte(strings.TrimSpace(userAgent)))
	payload := captchaTokenPayload{
		ChallengeID: challenge.ID,
		IP:          challenge.IP,
		UAHash:      uaHash,
		ExpiresAt:   now.Add(sliderTokenTTL).Unix(),
		Nonce:       utils.NewSessionID()[:16],
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sig := signCaptchaPayload(raw)
	return fmt.Sprintf("%s.%s",
		base64.RawURLEncoding.EncodeToString(raw),
		base64.RawURLEncoding.EncodeToString(sig),
	), nil
}

func signCaptchaPayload(raw []byte) []byte {
	mac := hmac.New(sha256.New, utils.JWTSecret)
	mac.Write(raw)
	return mac.Sum(nil)
}

func validateCaptchaToken(token, ip, userAgent string, now time.Time) (*sliderChallengeState, int, string) {
	loginConfigOnce.Do(loadLoginSecurityConfig)
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 {
		return nil, http.StatusBadRequest, "请先完成滑动验证码"
	}
	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, http.StatusBadRequest, "滑动验证码无效，请重试"
	}
	sigRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, http.StatusBadRequest, "滑动验证码无效，请重试"
	}
	expect := signCaptchaPayload(payloadRaw)
	if !hmac.Equal(expect, sigRaw) {
		return nil, http.StatusBadRequest, "滑动验证码无效，请重试"
	}

	var payload captchaTokenPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return nil, http.StatusBadRequest, "滑动验证码无效，请重试"
	}
	if payload.ExpiresAt <= now.Unix() {
		return nil, http.StatusBadRequest, "滑动验证码已过期，请重试"
	}
	if strings.TrimSpace(payload.Nonce) == "" {
		return nil, http.StatusBadRequest, "滑动验证码无效，请重试"
	}
	if payload.IP != ip {
		return nil, http.StatusBadRequest, "滑动验证码与当前设备不匹配"
	}
	if payload.UAHash == "" || payload.UAHash != sha256Hex([]byte(strings.TrimSpace(userAgent))) {
		return nil, http.StatusBadRequest, "滑动验证码与当前设备不匹配"
	}
	if ttl := time.Until(time.Unix(payload.ExpiresAt, 0)); ttl > 0 {
		ok, err := redisClient().SetNX(context.Background(), captchaNonceKey(payload.Nonce), "1", ttl).Result()
		if err != nil {
			return nil, http.StatusTooManyRequests, "登录防护繁忙，请稍后重试"
		}
		if !ok {
			return nil, http.StatusBadRequest, "滑动验证码已被使用，请刷新重试"
		}
	}
	if strings.TrimSpace(payload.ChallengeID) != "" {
		exists, err := redisClient().Exists(context.Background(), challengeConsumedKey(payload.ChallengeID)).Result()
		if err != nil {
			return nil, http.StatusTooManyRequests, "登录防护繁忙，请稍后重试"
		}
		if exists > 0 {
			return nil, http.StatusBadRequest, "滑动验证码已失效，请刷新重试"
		}
	}

	challenge, err := loadChallenge(payload.ChallengeID)
	if err != nil {
		return nil, http.StatusBadRequest, "滑动验证码已失效，请刷新重试"
	}
	if challenge == nil || challenge.IP != ip || !challenge.Verified || challenge.Consumed || now.After(challenge.ExpiresAt) {
		return nil, http.StatusBadRequest, "滑动验证码已失效，请刷新重试"
	}
	return challenge, http.StatusOK, ""
}

func consumeCaptchaChallenge(challengeID string) {
	if strings.TrimSpace(challengeID) == "" {
		return
	}
	ctx := context.Background()
	challenge, err := loadChallenge(challengeID)
	if err != nil || challenge == nil {
		_ = redisClient().Set(ctx, challengeConsumedKey(challengeID), "1", sliderChallengeTTL).Err()
		return
	}
	challenge.Consumed = true
	ttl := ttlFromNow(challenge.ExpiresAt, 30*time.Second)
	_ = saveChallenge(challenge, ttl)
	_ = redisClient().Set(ctx, challengeConsumedKey(challengeID), "1", ttl).Err()
}
