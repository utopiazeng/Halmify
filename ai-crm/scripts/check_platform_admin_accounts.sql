-- 检查 tenant_id=0 的平台账号身份是否符合 super_admin 标准
SELECT
  id,
  username,
  tenant_id,
  status,
  account_type,
  role
FROM users
WHERE tenant_id = 0
ORDER BY id ASC;

-- 异常候选（建议校准）
SELECT
  id,
  username,
  tenant_id,
  status,
  account_type,
  role
FROM users
WHERE tenant_id = 0
  AND status = 'active'
  AND (
    account_type IS NULL OR account_type = '' OR account_type NOT IN ('super_admin')
  )
ORDER BY id ASC;
