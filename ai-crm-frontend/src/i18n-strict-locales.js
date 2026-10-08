export const strictLocales = [
  'zh-CN', 'zh-TW', 'en-US', 'es-ES', 'fr-FR', 'de-DE',
  'ja-JP', 'ko-KR', 'pt-PT', 'ar-SA', 'ru-RU', 'it-IT'
]

export const strictTranslationMissingByLocale = {
  'zh-CN': '翻译缺失',
  'zh-TW': '翻譯缺失',
  'en-US': 'Missing translation',
  'es-ES': 'Falta traducción',
  'fr-FR': 'Traduction manquante',
  'de-DE': 'Fehlende Übersetzung',
  'ja-JP': '翻訳がありません',
  'ko-KR': '번역 없음',
  'pt-PT': 'Tradução em falta',
  'ar-SA': 'الترجمة مفقودة',
  'ru-RU': 'Отсутствует перевод',
  'it-IT': 'Traduzione mancante'
}

const nonHanLocales = new Set(['en-US', 'es-ES', 'fr-FR', 'de-DE', 'pt-PT', 'ar-SA', 'ru-RU', 'it-IT'])

const exact = {
  '查询': ['Search', 'Buscar', 'Rechercher', 'Suchen', '検索', '검색', 'Pesquisar', 'بحث', 'Поиск', 'Cerca', '查詢'],
  '重置': ['Reset', 'Restablecer', 'Réinitialiser', 'Zurücksetzen', 'リセット', '초기화', 'Repor', 'إعادة تعيين', 'Сбросить', 'Reimposta', '重設'],
  '新增': ['Create', 'Crear', 'Créer', 'Erstellen', '作成', '생성', 'Criar', 'إنشاء', 'Создать', 'Crea', '新增'],
  '编辑': ['Edit', 'Editar', 'Modifier', 'Bearbeiten', '編集', '편집', 'Editar', 'تحرير', 'Изменить', 'Modifica', '編輯'],
  '删除': ['Delete', 'Eliminar', 'Supprimer', 'Löschen', '削除', '삭제', 'Eliminar', 'حذف', 'Удалить', 'Elimina', '刪除'],
  '保存': ['Save', 'Guardar', 'Enregistrer', 'Speichern', '保存', '저장', 'Guardar', 'حفظ', 'Сохранить', 'Salva', '儲存'],
  '提交': ['Submit', 'Enviar', 'Soumettre', 'Absenden', '送信', '제출', 'Submeter', 'إرسال', 'Отправить', 'Invia', '提交'],
  '取消': ['Cancel', 'Cancelar', 'Annuler', 'Abbrechen', 'キャンセル', '취소', 'Cancelar', 'إلغاء', 'Отмена', 'Annulla', '取消'],
  '导出': ['Export', 'Exportar', 'Exporter', 'Exportieren', 'エクスポート', '내보내기', 'Exportar', 'تصدير', 'Экспорт', 'Esporta', '匯出'],
  '刷新': ['Refresh', 'Actualizar', 'Actualiser', 'Aktualisieren', '更新', '새로고침', 'Atualizar', 'تحديث', 'Обновить', 'Aggiorna', '重新整理'],
  '查看': ['View', 'Ver', 'Voir', 'Anzeigen', '表示', '보기', 'Ver', 'عرض', 'Просмотр', 'Visualizza', '查看'],
  '操作': ['Actions', 'Acciones', 'Actions', 'Aktionen', '操作', '작업', 'Ações', 'الإجراءات', 'Действия', 'Azioni', '操作'],
  '状态': ['Status', 'Estado', 'Statut', 'Status', 'ステータス', '상태', 'Estado', 'الحالة', 'Статус', 'Stato', '狀態'],
  '共': ['Total', 'Total', 'Total', 'Gesamt', '合計', '총', 'Total', 'الإجمالي', 'Всего', 'Totale', '共'],
  '批量推进到': ['Advance to', 'Avanzar a', 'Avancer vers', 'Verschieben nach', '次へ進める', '다음으로 이동', 'Avançar para', 'تقديم إلى', 'Перевести в', 'Avanza a', '批量推進到'],
  '概率': ['Probability', 'Probabilidad', 'Probabilité', 'Wahrscheinlichkeit', '確率', '확률', 'Probabilidade', 'الاحتمال', 'Вероятность', 'Probabilità', '機率'],
  '预计成交': ['Close date', 'Fecha cierre', 'Date clôture', 'Abschlussdatum', '成約予定日', '예상 체결일', 'Data fecho', 'تاريخ الإغلاق', 'Дата закрытия', 'Data chiusura', '預計成交'],
  '列设置': ['Columns', 'Columnas', 'Colonnes', 'Spalten', '列設定', '열 설정', 'Colunas', 'الأعمدة', 'Колонки', 'Colonne', '欄位設定'],
  '逾期': ['Overdue', 'Vencido', 'En retard', 'Überfällig', '期限超過', '기한 초과', 'Atrasado', 'متأخر', 'Просрочено', 'Scaduto', '逾期'],
  '内容': ['Message', 'Mensaje', 'Message', 'Nachricht', 'メッセージ', '메시지', 'Mensagem', 'الرسالة', 'Сообщение', 'Messaggio', '訊息'],
  '详情': ['Info', 'Info', 'Infos', 'Info', '情報', '정보', 'Info', 'معلومات', 'Инфо', 'Info', '資訊'],
  '基础信息': ['Basic', 'Básico', 'Base', 'Basis', '基本', '기본', 'Básico', 'أساسي', 'Основное', 'Base', '基本資訊'],
  '公司资料': ['Profile', 'Perfil', 'Profil', 'Profil', 'プロフィール', '프로필', 'Perfil', 'الملف', 'Профиль', 'Profilo', '公司資料'],
  '洞察与链接': ['Insights', 'Insights', 'Insights', 'Insights', 'インサイト', '인사이트', 'Insights', 'الرؤى', 'Инсайты', 'Insight', '洞察與連結'],
  '详情文本': ['Notes', 'Notas', 'Notes', 'Notizen', 'メモ', '메모', 'Notas', 'ملاحظات', 'Заметки', 'Note', '詳情文字'],
  '客户健康度': ['Health', 'Salud', 'Santé', 'Gesundheit', '健全性', '상태', 'Saúde', 'الصحة', 'Здоровье', 'Salute', '客戶健康度'],
  '公开商机': ['Public', 'Público', 'Public', 'Öffentlich', '公開', '공개', 'Público', 'عام', 'Публичный', 'Pubblico', '公開商機'],
  '私密商机': ['Private', 'Privado', 'Privé', 'Privat', '非公開', '비공개', 'Privado', 'خاص', 'Приватный', 'Privato', '私密商機'],
  '成交额': ['Revenue', 'Ingresos', 'Revenu', 'Umsatz', '売上', '매출', 'Receita', 'الإيراد', 'Выручка', 'Ricavi', '成交額'],
  '需求不符': ['Poor fit', 'No encaja', 'Peu adapté', 'Passt nicht', '不一致', '부적합', 'Pouco adequado', 'غير مناسب', 'Не подходит', 'Non adatto', '需求不符'],
  '销售开发代表': ['SDR', 'SDR', 'SDR', 'SDR', 'SDR', 'SDR', 'SDR', 'SDR', 'SDR', 'SDR', '銷售開發代表'],
  '财税设置': ['Tax settings', 'Ajustes fiscales', 'Réglages fiscaux', 'Steuereinstellungen', '税設定', '세무 설정', 'Definições fiscais', 'إعدادات الضريبة', 'Налоги', 'Impostazioni fiscali', '財稅設定'],
  '手机号': ['Mobile', 'Móvil', 'Mobile', 'Mobil', '携帯電話', '휴대폰', 'Telemóvel', 'الجوال', 'Мобильный', 'Cellulare', '手機號'],
  '汇率': ['FX rate', 'Tipo cambio', 'Taux change', 'Wechselkurs', '為替レート', '환율', 'Taxa câmbio', 'سعر الصرف', 'Курс', 'Cambio', '匯率'],
  '客户等级': ['Grade', 'Grado', 'Niveau', 'Stufe', 'ランク', '등급', 'Grau', 'الدرجة', 'Уровень', 'Grado', '客戶等級'],
  '线索来源': ['Source', 'Origen', 'Source', 'Quelle', 'ソース', '출처', 'Origem', 'المصدر', 'Источник', 'Fonte', '線索來源'],
  '可见范围': ['Visibility', 'Visibilidad', 'Visibilité', 'Sichtbarkeit', '表示範囲', '가시성', 'Visibilidade', 'الرؤية', 'Видимость', 'Visibilità', '可見範圍'],
  '标题': ['Title', 'Título', 'Titre', 'Titel', 'タイトル', '제목', 'Título', 'العنوان', 'Заголовок', 'Titolo', '標題'],
  '部门编码': ['Dept. code', 'Código depto.', 'Code service', 'Abteilungscode', '部署コード', '부서 코드', 'Código dep.', 'رمز القسم', 'Код отдела', 'Codice reparto', '部門編碼'],
  '官网': ['Website', 'Web', 'Site web', 'Website', 'Webサイト', '웹사이트', 'Site', 'الموقع', 'Сайт', 'Sito web', '官網'],
  '性别': ['Gender', 'Género', 'Genre', 'Geschlecht', '性別', '성별', 'Género', 'الجنس', 'Пол', 'Genere', '性別'],
  '数量': ['Qty', 'Cant.', 'Qté', 'Menge', '数量', '수량', 'Qtd.', 'الكمية', 'Кол-во', 'Qtà', '數量'],
  '单价': ['Price', 'Precio', 'Prix', 'Preis', '単価', '단가', 'Preço', 'السعر', 'Цена', 'Prezzo', '單價'],
  '折扣': ['Discount', 'Descuento', 'Remise', 'Rabatt', '割引', '할인', 'Desconto', 'خصم', 'Скидка', 'Sconto', '折扣'],
  '税率': ['Tax', 'Impuesto', 'Taxe', 'Steuer', '税率', '세율', 'Imposto', 'ضريبة', 'Налог', 'Imposta', '稅率'],
  '应收': ['Receivable', 'Por cobrar', 'À recevoir', 'Forderung', '未収', '미수', 'A receber', 'مستحق', 'К получению', 'Da incassare', '應收'],
  '已读': ['Read', 'Leído', 'Lu', 'Gelesen', '既読', '읽음', 'Lida', 'مقروء', 'Прочитано', 'Letto', '已讀'],
  '未读': ['Unread', 'No leído', 'Non lu', 'Ungelesen', '未読', '읽지 않음', 'Não lida', 'غير مقروء', 'Непрочитано', 'Non letto', '未讀'],
  '已归档': ['Archived', 'Archivado', 'Archivé', 'Archiviert', 'アーカイブ済み', '보관됨', 'Arquivada', 'مؤرشف', 'В архиве', 'Archiviato', '已歸檔'],
  '已取消': ['Canceled', 'Cancelado', 'Annulé', 'Storniert', 'キャンセル済み', '취소됨', 'Cancelado', 'ملغى', 'Отменено', 'Annullato', '已取消'],
  '已完成': ['Done', 'Listo', 'Terminé', 'Erledigt', '完了', '완료', 'Concluído', 'تم', 'Готово', 'Completato', '已完成'],
  '关键词': ['Keyword', 'Palabra clave', 'Mot-clé', 'Schlüsselwort', 'キーワード', '키워드', 'Palavra-chave', 'كلمة مفتاحية', 'Ключевое слово', 'Parola chiave', '關鍵字'],
  '姓名': ['Name', 'Nombre', 'Nom', 'Name', '氏名', '이름', 'Nome', 'الاسم', 'Имя', 'Nome', '姓名'],
  '公司': ['Company', 'Empresa', 'Entreprise', 'Unternehmen', '会社', '회사', 'Empresa', 'الشركة', 'Компания', 'Azienda', '公司'],
  '公司名': ['Company name', 'Nombre de empresa', 'Nom de l’entreprise', 'Firmenname', '会社名', '회사명', 'Nome da empresa', 'اسم الشركة', 'Название компании', 'Nome azienda', '公司名'],
  '手机': ['Mobile', 'Móvil', 'Mobile', 'Mobil', '携帯電話', '휴대폰', 'Telemóvel', 'الجوال', 'Мобильный', 'Cellulare', '手機'],
  '电话': ['Phone', 'Teléfono', 'Téléphone', 'Telefon', '電話', '전화', 'Telefone', 'الهاتف', 'Телефон', 'Telefono', '電話'],
  '邮箱': ['Email', 'Correo', 'E-mail', 'E-Mail', 'メール', '이메일', 'E-mail', 'البريد الإلكتروني', 'Email', 'Email', '信箱'],
  '负责人': ['Owner', 'Responsable', 'Responsable', 'Verantwortlich', '担当者', '담당자', 'Responsável', 'المسؤول', 'Ответственный', 'Responsabile', '負責人'],
  '创建时间': ['Created at', 'Creado el', 'Créé le', 'Erstellt am', '作成日時', '생성 시간', 'Criado em', 'وقت الإنشاء', 'Создано', 'Creato il', '建立時間'],
  '更新时间': ['Updated at', 'Actualizado el', 'Mis à jour le', 'Aktualisiert am', '更新日時', '수정 시간', 'Atualizado em', 'وقت التحديث', 'Обновлено', 'Aggiornato il', '更新時間'],
  '启用': ['Enabled', 'Activo', 'Activé', 'Aktiviert', '有効', '활성화', 'Ativo', 'مفعل', 'Включено', 'Attivo', '啟用'],
  '禁用': ['Disabled', 'Desactivado', 'Désactivé', 'Deaktiviert', '無効', '비활성화', 'Inativo', 'معطل', 'Отключено', 'Disattivato', '停用'],
  '在线': ['Online', 'En línea', 'En ligne', 'Online', 'オンライン', '온라인', 'Online', 'متصل', 'Онлайн', 'Online', '線上'],
  '离线': ['Offline', 'Desconectado', 'Hors ligne', 'Offline', 'オフライン', '오프라인', 'Offline', 'غير متصل', 'Офлайн', 'Offline', '離線'],
  '公开': ['Public', 'Público', 'Public', 'Öffentlich', '公開', '공개', 'Público', 'عام', 'Публичный', 'Pubblico', '公開'],
  '私密': ['Private', 'Privado', 'Privé', 'Privat', '非公開', '비공개', 'Privado', 'خاص', 'Приватный', 'Privato', '私密'],
  '用户': ['User', 'Usuario', 'Utilisateur', 'Benutzer', 'ユーザー', '사용자', 'Utilizador', 'المستخدم', 'Пользователь', 'Utente', '用戶'],
  '角色': ['Role', 'Rol', 'Rôle', 'Rolle', 'ロール', '역할', 'Função', 'الدور', 'Роль', 'Ruolo', '角色'],
  '客户': ['Customer', 'Cliente', 'Client', 'Kunde', '顧客', '고객', 'Cliente', 'العميل', 'Клиент', 'Cliente', '客戶'],
  '联系人': ['Contact', 'Contacto', 'Contact', 'Kontakt', '連絡先', '연락처', 'Contacto', 'جهة الاتصال', 'Контакт', 'Contatto', '聯絡人'],
  '对象': ['Object', 'Objeto', 'Objet', 'Objekt', '対象', '객체', 'Objeto', 'العنصر', 'Объект', 'Oggetto', '物件'],
  '项目': ['Project', 'Proyecto', 'Projet', 'Projekt', 'プロジェクト', '프로젝트', 'Projeto', 'مشروع', 'Проект', 'Progetto', '專案'],
  '线索': ['Lead', 'Lead', 'Prospect', 'Lead', 'リード', '리드', 'Lead', 'عميل محتمل', 'Лид', 'Lead', '線索'],
  '交易商机': ['Opportunity', 'Oportunidad', 'Opportunité', 'Verkaufschance', '商談', '영업기회', 'Oportunidade', 'فرصة', 'Сделка', 'Opportunità', '交易商機'],
  '商机': ['Opportunity', 'Oportunidad', 'Opportunité', 'Chance', '商談', '영업기회', 'Oportunidade', 'فرصة', 'Возможность', 'Opportunità', '商機'],
  '成交记录': ['Deal record', 'Registro de cierre', 'Affaire conclue', 'Abschlussdatensatz', '成約記録', '거래 기록', 'Registo de negócio', 'سجل الصفقة', 'Запись сделки', 'Record accordo', '成交記錄'],
  '成交': ['Deal', 'Trato', 'Affaire', 'Deal', '成約', '거래', 'Negócio', 'صفقة', 'Сделка', 'Accordo', '成交'],
  '订单': ['Order', 'Pedido', 'Commande', 'Auftrag', '注文', '주문', 'Encomenda', 'طلب', 'Заказ', 'Ordine', '訂單'],
  '客户项目': ['Customer project', 'Proyecto de cliente', 'Projet client', 'Kundenprojekt', '顧客プロジェクト', '고객 프로젝트', 'Projeto de cliente', 'مشروع العميل', 'Клиентский проект', 'Progetto cliente', '客戶專案'],
  '服务工单': ['Service ticket', 'Ticket de servicio', 'Ticket service', 'Service-Ticket', 'サービスチケット', '서비스 티켓', 'Pedido de serviço', 'تذكرة خدمة', 'Сервисная заявка', 'Ticket servizio', '服務工單'],
  '市场活动': ['Campaign', 'Campaña', 'Campagne', 'Kampagne', 'キャンペーン', '캠페인', 'Campanha', 'حملة', 'Кампания', 'Campagna', '市場活動'],
  '销售大屏': ['Sales dashboard', 'Panel de ventas', 'Tableau de ventes', 'Vertriebsdashboard', '営業ダッシュボード', '영업 대시보드', 'Painel de vendas', 'لوحة المبيعات', 'Панель продаж', 'Dashboard vendite', '銷售大屏'],
  '账号权限': ['Account permissions', 'Permisos de cuenta', 'Droits des comptes', 'Kontoberechtigungen', 'アカウント権限', '계정 권한', 'Permissões de conta', 'صلاحيات الحساب', 'Права аккаунта', 'Permessi account', '帳號權限'],
  '团队管理': ['Team management', 'Gestión de equipos', 'Gestion d’équipe', 'Teamverwaltung', 'チーム管理', '팀 관리', 'Gestão de equipas', 'إدارة الفريق', 'Управление командой', 'Gestione team', '團隊管理'],
  '产品服务': ['Products and services', 'Productos y servicios', 'Produits et services', 'Produkte und Services', '製品サービス', '제품 서비스', 'Produtos e serviços', 'المنتجات والخدمات', 'Продукты и услуги', 'Prodotti e servizi', '產品服務'],
  '企业管理': ['Enterprise management', 'Gestión empresarial', 'Gestion d’entreprise', 'Unternehmensverwaltung', '企業管理', '기업 관리', 'Gestão empresarial', 'إدارة المؤسسة', 'Управление предприятием', 'Gestione aziendale', '企業管理'],
  '工作区': ['Workspace', 'Área de trabajo', 'Espace de travail', 'Arbeitsbereich', 'ワークスペース', '작업공간', 'Área de trabalho', 'مساحة العمل', 'Рабочая область', 'Area di lavoro', '工作區']
}

Object.assign(exact, {
  '会社電話': ['Company phone', 'Teléfono empresa', 'Téléphone société', 'Firmentelefon', '会社電話', '회사 전화', 'Telefone da empresa', 'هاتف الشركة', 'Телефон компании', 'Telefono azienda', '公司電話'],
  '添加明细': ['Add item', 'Añadir línea', 'Ajouter ligne', 'Position hinzufügen', '明細を追加', '항목 추가', 'Adicionar item', 'إضافة بند', 'Добавить строку', 'Aggiungi riga', '新增明細'],
  '有效期': ['Valid until', 'Válido hasta', 'Valide jusqu’au', 'Gültig bis', '有効期限', '유효기간', 'Válido até', 'صالح حتى', 'Действует до', 'Valido fino', '有效期'],
  '培训安排': ['Training', 'Formación', 'Formation', 'Schulung', '研修', '교육', 'Formação', 'التدريب', 'Обучение', 'Formazione', '培訓安排'],
  '· 折扣': [' · discount', ' · descuento', ' · remise', ' · Rabatt', '・割引', ' · 할인', ' · desconto', ' · خصم', ' · скидка', ' · sconto', ' · 折扣'],
  '所属模块': ['Module', 'Módulo', 'Module', 'Modul', 'モジュール', '모듈', 'Módulo', 'الوحدة', 'Модуль', 'Modulo', '所屬模組'],
  '行': ['Row', 'Fila', 'Ligne', 'Zeile', '行', '행', 'Linha', 'صف', 'Строка', 'Riga', '列'],
  '执行记录': ['Execution', 'Ejecución', 'Exécution', 'Ausführung', '実行記録', '실행 기록', 'Execução', 'تنفيذ', 'Выполнение', 'Esecuzione', '執行記錄'],
  '多个链接用逗号分隔': ['Separate links with commas', 'Separa enlaces con comas', 'Séparez par des virgules', 'Links mit Kommas trennen', 'リンクはカンマ区切り', '쉼표로 링크 구분', 'Separe links por vírgulas', 'افصل الروابط بفواصل', 'Ссылки через запятую', 'Link separati da virgole', '多個連結以逗號分隔'],
  '升级': ['Escalate', 'Escalar', 'Escalader', 'Eskalieren', 'エスカレーション', '에스컬레이션', 'Escalar', 'تصعيد', 'Эскалация', 'Escala', '升級'],
  '评分': ['Score', 'Puntuación', 'Score', 'Score', 'スコア', '점수', 'Pontuação', 'النقاط', 'Оценка', 'Punteggio', '評分'],
  '信息无效': ['Invalid info', 'Info no válida', 'Info invalide', 'Ungültige Info', '無効な情報', '잘못된 정보', 'Info inválida', 'معلومات غير صالحة', 'Неверные данные', 'Info non valida', '資訊無效'],
  '转派给': ['Transfer to', 'Transferir a', 'Transférer à', 'Weiterleiten an', '転送先', '전달 대상', 'Transferir para', 'تحويل إلى', 'Передать', 'Trasferisci a', '轉派給'],
  '会议': ['Meeting', 'Reunión', 'Réunion', 'Meeting', '会議', '회의', 'Reunião', 'اجتماع', 'Встреча', 'Riunione', '會議'],
  '再营销': ['Remarketing', 'Remarketing', 'Remarketing', 'Remarketing', 'リマーケティング', '리마케팅', 'Remarketing', 'إعادة التسويق', 'Ремаркетинг', 'Remarketing', '再行銷'],
  '服务交付': ['Delivery', 'Entrega', 'Livraison', 'Lieferung', '提供', '서비스 제공', 'Entrega', 'التسليم', 'Доставка', 'Consegna', '服務交付'],
  '较急': ['Urgent', 'Urgente', 'Urgent', 'Dringend', '緊急', '긴급', 'Urgente', 'عاجل', 'Срочно', 'Urgente', '較急'],
  '关闭评价': ['Close review', 'Cerrar reseña', 'Clore avis', 'Bewertung schließen', '評価を閉じる', '평가 닫기', 'Fechar avaliação', 'إغلاق التقييم', 'Закрыть отзыв', 'Chiudi valutazione', '關閉評價'],
  '验收标准': ['Acceptance', 'Aceptación', 'Acceptation', 'Abnahme', '受入基準', '검수 기준', 'Aceitação', 'القبول', 'Приемка', 'Accettazione', '驗收標準'],
  '预警线': ['Alert line', 'Línea alerta', 'Seuil alerte', 'Warnschwelle', '警告ライン', '경고선', 'Linha alerta', 'حد التنبيه', 'Порог', 'Soglia', '預警線'],
  '等': ['etc.', 'etc.', 'etc.', 'usw.', '等', '등', 'etc.', 'إلخ', 'и др.', 'ecc.', '等'],
  '按成交额': ['By revenue', 'Por ingresos', 'Par revenu', 'Nach Umsatz', '売上順', '매출 기준', 'Por receita', 'حسب الإيراد', 'По выручке', 'Per ricavi', '按成交額'],
  '全选': ['Select all', 'Seleccionar todo', 'Tout sélectionner', 'Alle auswählen', '全選択', '전체 선택', 'Selecionar tudo', 'تحديد الكل', 'Выбрать все', 'Seleziona tutto', '全選'],
  '意见反馈': ['Feedback', 'Comentarios', 'Retour', 'Feedback', 'フィードバック', '피드백', 'Feedback', 'ملاحظات', 'Обратная связь', 'Feedback', '意見回饋'],
  '还有明细未发': ['Items not shipped', 'Líneas sin enviar', 'Lignes non expédiées', 'Positionen offen', '未発送明細あり', '미발송 항목 있음', 'Itens por enviar', 'بنود غير مرسلة', 'Есть неотгруженные', 'Righe non spedite', '尚有明細未發'],
  '我该关注什么': ['Focus', 'Enfoque', 'Priorité', 'Fokus', '注目点', '집중 항목', 'Foco', 'التركيز', 'Фокус', 'Focus', '我該關注什麼'],
  '外贸服务': ['Trade services', 'Servicios comercio', 'Services commerce', 'Handelsservices', '貿易サービス', '무역 서비스', 'Serviços comércio', 'خدمات التجارة', 'ВЭД-услуги', 'Servizi commercio', '外貿服務'],
  '目录': ['Catalog', 'Catálogo', 'Catalogue', 'Katalog', 'カタログ', '카탈로그', 'Catálogo', 'الفهرس', 'Каталог', 'Catalogo', '目錄'],
  '未留联系方式': ['No contact info', 'Sin contacto', 'Sans contact', 'Keine Kontaktdaten', '連絡先なし', '연락처 없음', 'Sem contacto', 'لا توجد جهة اتصال', 'Нет контактов', 'Senza contatto', '未留聯絡方式'],
  '已发货': ['Shipped', 'Enviado', 'Expédié', 'Versandt', '発送済み', '배송됨', 'Enviado', 'تم الشحن', 'Отгружено', 'Spedito', '已發貨'],
  '上次执行': ['Last run', 'Última ejecución', 'Dernière exécution', 'Letzte Ausführung', '前回実行', '마지막 실행', 'Última execução', 'آخر تشغيل', 'Последний запуск', 'Ultima esecuzione', '上次執行'],
  '分类': ['Category', 'Categoría', 'Catégorie', 'Kategorie', 'カテゴリ', '분류', 'Categoria', 'الفئة', 'Категория', 'Categoria', '分類'],
  '滑块验证码': ['Slider captcha', 'Captcha deslizante', 'Captcha curseur', 'Slider-Captcha', 'スライダー認証', '슬라이더 인증', 'Captcha deslizante', 'تحقق بالسحب', 'Слайдер-капча', 'Captcha slider', '滑塊驗證碼'],
  '操作地址': ['IP address', 'Dirección IP', 'Adresse IP', 'IP-Adresse', 'IPアドレス', 'IP 주소', 'Endereço IP', 'عنوان IP', 'IP-адрес', 'Indirizzo IP', '操作位址'],
  '已转换': ['Converted', 'Convertido', 'Converti', 'Konvertiert', '変換済み', '전환됨', 'Convertido', 'تم التحويل', 'Конвертировано', 'Convertito', '已轉換'],
  '原因排行': ['Top reasons', 'Motivos principales', 'Raisons clés', 'Top-Gründe', '理由ランキング', '사유 순위', 'Principais motivos', 'أهم الأسباب', 'Топ причин', 'Motivi principali', '原因排行'],
  '摘要': ['Summary', 'Resumen', 'Résumé', 'Zusammenfassung', '概要', '요약', 'Resumo', 'ملخص', 'Сводка', 'Riepilogo', '摘要'],
  '互动': ['Interaction', 'Interacción', 'Interaction', 'Interaktion', 'やり取り', '상호작용', 'Interação', 'تفاعل', 'Взаимодействие', 'Interazione', '互動'],
  '成交趋势图': ['Deal trend', 'Tendencia tratos', 'Tendance affaires', 'Deal-Trend', '成約トレンド', '거래 추세', 'Tendência negócios', 'اتجاه الصفقات', 'Тренд сделок', 'Trend accordi', '成交趨勢圖'],
  '管理': ['Manage', 'Gestionar', 'Gérer', 'Verwalten', '管理', '관리', 'Gerir', 'إدارة', 'Управлять', 'Gestisci', '管理'],
  '文件': ['Files', 'Archivos', 'Fichiers', 'Dateien', 'ファイル', '파일', 'Ficheiros', 'ملفات', 'Файлы', 'File', '檔案'],
  '马上': ['Now', 'Ahora', 'Maintenant', 'Jetzt', '今すぐ', '지금', 'Agora', 'الآن', 'Сейчас', 'Ora', '馬上'],
  '使用者': ['User', 'Usuario', 'Utilisateur', 'Benutzer', 'ユーザー', '사용자', 'Utilizador', 'المستخدم', 'Пользователь', 'Utente', '使用者'],
  '单次上限': ['Single limit', 'Límite único', 'Limite unitaire', 'Einzellimit', '単回上限', '1회 한도', 'Limite único', 'حد المرة', 'Разовый лимит', 'Limite singolo', '單次上限'],
  '销售预测': ['Forecast', 'Previsión', 'Prévision', 'Prognose', '予測', '예측', 'Previsão', 'توقعات', 'Прогноз', 'Previsione', '銷售預測'],
  '推断': ['Inferred', 'Inferido', 'Déduit', 'Abgeleitet', '推定', '추정됨', 'Inferido', 'مستنتج', 'Выведено', 'Dedotto', '推斷'],
  '触达方式': ['Channel', 'Canal', 'Canal', 'Kanal', '接点', '채널', 'Canal', 'القناة', 'Канал', 'Canale', '觸達方式'],
  '退出大屏': ['Exit dashboard', 'Salir panel', 'Quitter tableau', 'Dashboard verlassen', '終了', '대시보드 나가기', 'Sair painel', 'خروج من اللوحة', 'Выйти', 'Esci dashboard', '退出大屏'],
  '天后结束': ['days left', 'días restantes', 'jours restants', 'Tage übrig', '日後終了', '일 남음', 'dias restantes', 'أيام متبقية', 'дн. осталось', 'giorni rimasti', '天後結束'],
  '生物醫藥': ['Biopharma', 'Biofarma', 'Biopharma', 'Biopharma', 'バイオ医薬', '바이오제약', 'Biofarma', 'الأدوية الحيوية', 'Биофарма', 'Biofarmaceutico', '生物醫藥'],
  '服務': ['Service', 'Servicio', 'Service', 'Service', 'サービス', '서비스', 'Serviço', 'الخدمة', 'Сервис', 'Servizio', '服務'],
  '未命名产品': ['Unnamed product', 'Producto sin nombre', 'Produit sans nom', 'Unbenanntes Produkt', '未命名製品', '이름 없는 제품', 'Produto sem nome', 'منتج بلا اسم', 'Безымянный продукт', 'Prodotto senza nome', '未命名產品'],
  '專業服務': ['Professional services', 'Servicios profesionales', 'Services pros', 'Professional Services', '専門サービス', '전문 서비스', 'Serviços profissionais', 'خدمات مهنية', 'Проф. услуги', 'Servizi professionali', '專業服務'],
  '头像': ['Avatar', 'Avatar', 'Avatar', 'Avatar', 'アバター', '아바타', 'Avatar', 'الصورة', 'Аватар', 'Avatar', '頭像'],
  '素材资料': ['Materials', 'Materiales', 'Supports', 'Materialien', '素材', '자료', 'Materiais', 'المواد', 'Материалы', 'Materiali', '素材資料'],
  '转派总数': ['Transfers', 'Transferencias', 'Transferts', 'Weiterleitungen', '転送数', '전달 수', 'Transferências', 'التحويلات', 'Передачи', 'Trasferimenti', '轉派總數'],
  '应收余额': ['Receivable', 'Por cobrar', 'À recevoir', 'Forderung', '未収金', '미수금', 'A receber', 'مستحقات', 'Дебиторка', 'Da incassare', '應收餘額'],
  '把专业缩写翻译成日常动作': ['Explain terms in plain words', 'Explica términos simples', 'Expliquez simplement', 'Begriffe einfach erklären', '用語をわかりやすく説明', '용어를 쉽게 설명', 'Explique termos simples', 'اشرح المصطلحات ببساطة', 'Объяснить термины', 'Spiega i termini', '把專業縮寫翻成日常動作'],
  '已修复': ['Fixed', 'Corregido', 'Corrigé', 'Behoben', '修正済み', '수정됨', 'Corrigido', 'تم الإصلاح', 'Исправлено', 'Corretto', '已修復'],
  '标记已读': ['Mark read', 'Marcar leído', 'Marquer lu', 'Als gelesen markieren', '既読にする', '읽음 처리', 'Marcar lida', 'وضع كمقروء', 'Отметить прочитанным', 'Segna letto', '標記已讀'],
  '回款': ['Payment', 'Pago', 'Paiement', 'Zahlung', '入金', '수금', 'Pagamento', 'التحصيل', 'Платеж', 'Pagamento', '回款'],
  '男': ['Male', 'Hombre', 'Homme', 'Männlich', '男性', '남성', 'Masculino', 'ذكر', 'Мужской', 'Maschio', '男'],
  '营收区间': ['Revenue range', 'Rango ingresos', 'Fourchette CA', 'Umsatzspanne', '売上範囲', '매출 구간', 'Faixa receita', 'نطاق الإيراد', 'Диапазон выручки', 'Fascia ricavi', '營收區間'],
  '发现': ['Found', 'Encontrado', 'Trouvé', 'Gefunden', '検出', '발견', 'Encontrado', 'تم العثور', 'Найдено', 'Trovato', '發現'],
  '· 滞留': [' · stayed', ' · retenido', ' · bloqué', ' · verweilt', '・滞留', ' · 체류', ' · parado', ' · متوقف', ' · задержка', ' · fermo', ' · 滯留'],
  '工業設備': ['Industrial equipment', 'Equipo industrial', 'Équipement industriel', 'Industrieausrüstung', '産業機器', '산업 장비', 'Equipamento industrial', 'معدات صناعية', 'Пром. оборудование', 'Attrezzature industriali', '工業設備'],
  '添加一期': ['Add phase', 'Añadir fase', 'Ajouter phase', 'Phase hinzufügen', 'フェーズ追加', '단계 추가', 'Adicionar fase', 'إضافة مرحلة', 'Добавить этап', 'Aggiungi fase', '新增一期'],
  '可选': ['Optional', 'Opcional', 'Optionnel', 'Optional', '任意', '선택', 'Opcional', 'اختياري', 'Необязательно', 'Opzionale', '可選'],
  '角色名称': ['Role name', 'Nombre del rol', 'Nom du rôle', 'Rollenname', 'ロール名', '역할 이름', 'Nome da função', 'اسم الدور', 'Название роли', 'Nome ruolo', '角色名稱'],
  '角色编码': ['Role code', 'Código de rol', 'Code rôle', 'Rollencode', 'ロールコード', '역할 코드', 'Código da função', 'رمز الدور', 'Код роли', 'Codice ruolo', '角色編碼'],
  '角色描述': ['Description', 'Descripción', 'Description', 'Beschreibung', '説明', '설명', 'Descrição', 'الوصف', 'Описание', 'Descrizione', '角色描述'],
  '角色状态': ['Status', 'Estado', 'Statut', 'Status', 'ステータス', '상태', 'Estado', 'الحالة', 'Статус', 'Stato', '角色狀態'],
  '创建日期': ['Created', 'Creado', 'Créé', 'Erstellt', '作成日', '생성일', 'Criado', 'تم الإنشاء', 'Создано', 'Creato', '建立日期'],
  '编辑日期': ['Updated', 'Actualizado', 'Mis à jour', 'Aktualisiert', '更新日', '수정일', 'Atualizado', 'تم التحديث', 'Обновлено', 'Aggiornato', '更新日期'],
  '展开': ['Expand', 'Expandir', 'Développer', 'Erweitern', '展開', '펼치기', 'Expandir', 'توسيع', 'Развернуть', 'Espandi', '展開'],
  '收起': ['Collapse', 'Contraer', 'Réduire', 'Einklappen', '折りたたむ', '접기', 'Recolher', 'طي', 'Свернуть', 'Comprimi', '收合'],
  '保存当前筛选': ['Save view', 'Guardar vista', 'Enregistrer vue', 'Ansicht speichern', 'ビュー保存', '보기 저장', 'Guardar vista', 'حفظ العرض', 'Сохранить вид', 'Salva vista', '儲存檢視'],
  '已作废': ['Invalid', 'No válido', 'Invalide', 'Ungültig', '無効', '무효', 'Inválido', 'غير صالح', 'Недействительно', 'Non valido', '已作廢'],
  '作废': ['Invalidate', 'Invalidar', 'Invalider', 'Ungültig setzen', '無効化', '무효 처리', 'Invalidar', 'إبطال', 'Аннулировать', 'Invalida', '作廢'],
  'ABM名单': ['ABM list', 'Lista ABM', 'Liste ABM', 'ABM-Liste', 'ABMリスト', 'ABM 목록', 'Lista ABM', 'قائمة ABM', 'Список ABM', 'Lista ABM', 'ABM 名單'],
  '批量分配': ['Assign', 'Asignar', 'Assigner', 'Zuweisen', '割り当て', '배정', 'Atribuir', 'تعيين', 'Назначить', 'Assegna', '分配'],
  '批量作废': ['Invalidate', 'Invalidar', 'Invalider', 'Ungültig setzen', '無効化', '무효 처리', 'Invalidar', 'إبطال', 'Аннулировать', 'Invalida', '作廢'],
  '热度': ['Interest', 'Interés', 'Intérêt', 'Interesse', '関心度', '관심도', 'Interesse', 'الاهتمام', 'Интерес', 'Interesse', '熱度'],
  '已转商机': ['Converted', 'Convertido', 'Converti', 'Konvertiert', '変換済み', '전환됨', 'Convertido', 'تم التحويل', 'Конвертировано', 'Convertito', '已轉商機'],
  '筛选': ['Filter', 'Filtro', 'Filtre', 'Filter', 'フィルター', '필터', 'Filtro', 'عامل تصفية', 'Фильтр', 'Filtro', '篩選'],
  '批量': ['Batch', 'Lote', 'Lot', 'Stapel', '一括', '일괄', 'Lote', 'دفعة', 'Пакет', 'Batch', '批次'],
  '编码': ['Code', 'Código', 'Code', 'Code', 'コード', '코드', 'Código', 'الرمز', 'Код', 'Codice', '編碼'],
  '描述': ['Description', 'Descripción', 'Description', 'Beschreibung', '説明', '설명', 'Descrição', 'الوصف', 'Описание', 'Descrizione', '描述'],
  '日期': ['Date', 'Fecha', 'Date', 'Datum', '日付', '날짜', 'Data', 'التاريخ', 'Дата', 'Data', '日期'],
  '地区/规模': ['Region / size', 'Región / tamaño', 'Région / taille', 'Region / Größe', '地域 / 規模', '지역 / 규모', 'Região / dimensão', 'المنطقة / الحجم', 'Регион / размер', 'Regione / dimensione', '地區/規模'],
  '行业/细分': ['Industry / segment', 'Sector / segmento', 'Secteur / segment', 'Branche / Segment', '業界 / セグメント', '업종 / 세그먼트', 'Setor / segmento', 'القطاع / الشريحة', 'Отрасль / сегмент', 'Settore / segmento', '產業/細分'],
  '总部/规模': ['HQ / size', 'Sede / tamaño', 'Siège / taille', 'Zentrale / Größe', '本社 / 規模', '본사 / 규모', 'Sede / dimensão', 'المقر / الحجم', 'Штаб / размер', 'Sede / dimensione', '總部/規模'],
  '采购/增长信号': ['Buying / growth signals', 'Señales de compra / crecimiento', 'Signaux achat / croissance', 'Kauf- / Wachstumssignale', '購買 / 成長シグナル', '구매 / 성장 신호', 'Sinais de compra / crescimento', 'إشارات الشراء / النمو', 'Сигналы покупки / роста', 'Segnali acquisto / crescita', '採購/成長信號'],
  '风险/待核实': ['Risks / to verify', 'Riesgos / por verificar', 'Risques / à vérifier', 'Risiken / zu prüfen', 'リスク / 要確認', '리스크 / 확인 필요', 'Riscos / a verificar', 'المخاطر / للتحقق', 'Риски / проверить', 'Rischi / da verificare', '風險/待核實'],
  '电子/半导体': ['Electronics / semiconductor', 'Electrónica / semiconductores', 'Électronique / semi-conducteurs', 'Elektronik / Halbleiter', '電子 / 半導体', '전자 / 반도체', 'Eletrónica / semicondutores', 'الإلكترونيات / أشباه الموصلات', 'Электроника / полупроводники', 'Elettronica / semiconduttori', '電子/半導體'],
  '電子/半導體': ['Electronics / semiconductor', 'Electrónica / semiconductores', 'Électronique / semi-conducteurs', 'Elektronik / Halbleiter', '電子 / 半導体', '전자 / 반도체', 'Eletrónica / semicondutores', 'الإلكترونيات / أشباه الموصلات', 'Электроника / полупроводники', 'Elettronica / semiconduttori', '電子/半導體'],
  '汽车/零部件': ['Auto / parts', 'Automoción / piezas', 'Auto / pièces', 'Automobil / Teile', '自動車 / 部品', '자동차 / 부품', 'Automóvel / peças', 'السيارات / القطع', 'Авто / запчасти', 'Auto / ricambi', '汽車/零組件'],
  '汽車/零組件': ['Auto / parts', 'Automoción / piezas', 'Auto / pièces', 'Automobil / Teile', '自動車 / 部品', '자동차 / 부품', 'Automóvel / peças', 'السيارات / القطع', 'Авто / запчасти', 'Auto / ricambi', '汽車/零組件'],
  '建筑/工程': ['Construction / engineering', 'Construcción / ingeniería', 'BTP / ingénierie', 'Bau / Engineering', '建設 / エンジニアリング', '건설 / 엔지니어링', 'Construção / engenharia', 'البناء / الهندسة', 'Строительство / инжиниринг', 'Costruzioni / ingegneria', '建築/工程'],
  '建築/工程': ['Construction / engineering', 'Construcción / ingeniería', 'BTP / ingénierie', 'Bau / Engineering', '建設 / エンジニアリング', '건설 / 엔지니어링', 'Construção / engenharia', 'البناء / الهندسة', 'Строительство / инжиниринг', 'Costruzioni / ingegneria', '建築/工程'],
  '法律/财税': ['Legal / tax', 'Legal / fiscal', 'Juridique / fiscal', 'Recht / Steuern', '法務 / 税務', '법무 / 세무', 'Jurídico / fiscal', 'القانون / الضرائب', 'Право / налоги', 'Legale / fiscale', '法律/財稅'],
  '法律/財稅': ['Legal / tax', 'Legal / fiscal', 'Juridique / fiscal', 'Recht / Steuern', '法務 / 税務', '법무 / 세무', 'Jurídico / fiscal', 'القانون / الضرائب', 'Право / налоги', 'Legale / fiscale', '法律/財稅'],
  '广告/传媒': ['Advertising / media', 'Publicidad / medios', 'Publicité / médias', 'Werbung / Medien', '広告 / メディア', '광고 / 미디어', 'Publicidade / media', 'الإعلانات / الإعلام', 'Реклама / медиа', 'Pubblicità / media', '廣告/媒體'],
  '廣告/媒體': ['Advertising / media', 'Publicidad / medios', 'Publicité / médias', 'Werbung / Medien', '広告 / メディア', '광고 / 미디어', 'Publicidade / media', 'الإعلانات / الإعلام', 'Реклама / медиа', 'Pubblicità / media', '廣告/媒體'],
  '政府/非营利': ['Government / nonprofit', 'Gobierno / sin ánimo de lucro', 'Gouvernement / associatif', 'Regierung / Nonprofit', '政府 / 非営利', '정부 / 비영리', 'Governo / sem fins lucrativos', 'الحكومة / غير ربحي', 'Госсектор / НКО', 'Governo / nonprofit', '政府/非營利'],
  '政府/非營利': ['Government / nonprofit', 'Gobierno / sin ánimo de lucro', 'Gouvernement / associatif', 'Regierung / Nonprofit', '政府 / 非営利', '정부 / 비영리', 'Governo / sem fins lucrativos', 'الحكومة / غير ربحي', 'Госсектор / НКО', 'Governo / nonprofit', '政府/非營利'],
  '存储增购/续费': ['Storage add-on / renewal', 'Ampliación / renovación de almacenamiento', 'Stockage additionnel / renouvellement', 'Speichererweiterung / Verlängerung', 'ストレージ追加 / 更新', '스토리지 추가 / 갱신', 'Armazenamento extra / renovação', 'إضافة / تجديد التخزين', 'Расширение / продление хранилища', 'Storage aggiuntivo / rinnovo', '儲存增購/續費'],
  '电子邮件、短信、站内信': ['Email, SMS, in-app message', 'Correo, SMS, mensaje interno', 'E-mail, SMS, message interne', 'E-Mail, SMS, In-App-Nachricht', 'メール、SMS、アプリ内通知', '이메일, SMS, 앱 내 메시지', 'Email, SMS, mensagem interna', 'البريد، الرسائل، رسالة داخلية', 'Email, SMS, внутреннее сообщение', 'Email, SMS, messaggio interno', '電子郵件、簡訊、站內信'],
  '报名、直播、在线会议': ['Registration, live stream, online meeting', 'Registro, directo, reunión online', 'Inscription, live, réunion en ligne', 'Anmeldung, Livestream, Online-Meeting', '登録、ライブ、オンライン会議', '등록, 라이브, 온라인 회의', 'Inscrição, direto, reunião online', 'تسجيل، بث مباشر، اجتماع عبر الإنترنت', 'Регистрация, эфир, онлайн-встреча', 'Registrazione, live, riunione online', '報名、直播、線上會議'],
  '展会、会议、沙龙': ['Expo, meeting, salon', 'Feria, reunión, salón', 'Salon, réunion, événement', 'Messe, Meeting, Salon', '展示会、会議、サロン', '전시회, 회의, 살롱', 'Feira, reunião, salão', 'معرض، اجتماع، صالون', 'Выставка, встреча, салон', 'Fiera, meeting, salone', '展會、會議、沙龍'],
  '社交主页、帖子、私信': ['Social profile, posts, DMs', 'Perfil social, publicaciones, mensajes', 'Profil social, publications, messages', 'Social-Profil, Beiträge, DMs', 'SNSプロフィール、投稿、DM', '소셜 프로필, 게시물, DM', 'Perfil social, publicações, mensagens', 'ملف اجتماعي، منشورات، رسائل', 'Соцпрофиль, посты, личные сообщения', 'Profilo social, post, messaggi', '社交主頁、貼文、私訊'],
  '雲端運算/資料服務': ['Cloud / data services', 'Cloud / servicios de datos', 'Cloud / services de données', 'Cloud / Datendienste', 'クラウド / データサービス', '클라우드 / 데이터 서비스', 'Cloud / serviços de dados', 'السحابة / خدمات البيانات', 'Облако / сервисы данных', 'Cloud / servizi dati', '雲端運算/資料服務'],
  '代表電話、交換台、会社連絡先': ['Main phone, switchboard, or company contact', 'Teléfono principal, centralita o contacto de empresa', 'Téléphone principal, standard ou contact entreprise', 'Hauptnummer, Zentrale oder Unternehmenskontakt', '代表電話、交換台、会社連絡先', '대표 전화, 교환대 또는 회사 연락처', 'Telefone principal, central ou contacto da empresa', 'الهاتف الرئيسي أو البدالة أو جهة اتصال الشركة', 'Основной телефон, коммутатор или контакт компании', 'Telefono principale, centralino o contatto aziendale', '代表電話、交換台、公司聯絡方式'],
  '产品/服务明细': ['Product / service details', 'Detalles de producto / servicio', 'Détails produit / service', 'Produkt- / Servicedetails', '製品 / サービス詳細', '제품 / 서비스 상세', 'Detalhes de produto / serviço', 'تفاصيل المنتج / الخدمة', 'Сведения о продукте / услуге', 'Dettagli prodotto / servizio', '產品/服務明細'],
  '/ 预警': ['/ warning', '/ alerta', '/ alerte', '/ Warnung', '/ 警告', '/ 경고', '/ alerta', '/ تحذير', '/ предупреждение', '/ avviso', '/ 預警'],
  '产品/服务名称': ['Product / service name', 'Nombre de producto / servicio', 'Nom produit / service', 'Produkt- / Servicename', '製品 / サービス名', '제품 / 서비스명', 'Nome de produto / serviço', 'اسم المنتج / الخدمة', 'Название продукта / услуги', 'Nome prodotto / servizio', '產品/服務名稱']
})

const localeOrder = ['en-US', 'es-ES', 'fr-FR', 'de-DE', 'ja-JP', 'ko-KR', 'pt-PT', 'ar-SA', 'ru-RU', 'it-IT', 'zh-TW']
const exactByLocale = Object.fromEntries(strictLocales.map((locale) => [locale, {}]))
for (const [zh, values] of Object.entries(exact)) {
  exactByLocale['zh-CN'][zh] = zh
  localeOrder.forEach((locale, index) => {
    exactByLocale[locale][zh] = values[index]
  })
}

const templates = {
  enter: {
    'en-US': 'Enter {value}', 'es-ES': 'Introduce {value}', 'fr-FR': 'Saisir {value}', 'de-DE': '{value} eingeben',
    'ja-JP': '{value}を入力', 'ko-KR': '{value} 입력', 'pt-PT': 'Introduza {value}', 'ar-SA': 'أدخل {value}',
    'ru-RU': 'Введите {value}', 'it-IT': 'Inserisci {value}', 'zh-TW': '請輸入{value}'
  },
  select: {
    'en-US': 'Select {value}', 'es-ES': 'Selecciona {value}', 'fr-FR': 'Sélectionner {value}', 'de-DE': '{value} auswählen',
    'ja-JP': '{value}を選択', 'ko-KR': '{value} 선택', 'pt-PT': 'Selecione {value}', 'ar-SA': 'اختر {value}',
    'ru-RU': 'Выберите {value}', 'it-IT': 'Seleziona {value}', 'zh-TW': '請選擇{value}'
  },
  noData: {
    'en-US': 'No {value}', 'es-ES': 'Sin {value}', 'fr-FR': 'Aucun {value}', 'de-DE': 'Keine {value}',
    'ja-JP': '{value}なし', 'ko-KR': '{value} 없음', 'pt-PT': 'Sem {value}', 'ar-SA': 'لا يوجد {value}',
    'ru-RU': 'Нет {value}', 'it-IT': 'Nessun {value}', 'zh-TW': '暫無{value}'
  },
  create: {
    'en-US': 'Create {value}', 'es-ES': 'Crear {value}', 'fr-FR': 'Créer {value}', 'de-DE': '{value} erstellen',
    'ja-JP': '{value}を作成', 'ko-KR': '{value} 생성', 'pt-PT': 'Criar {value}', 'ar-SA': 'إنشاء {value}',
    'ru-RU': 'Создать {value}', 'it-IT': 'Crea {value}', 'zh-TW': '新增{value}'
  },
  edit: {
    'en-US': 'Edit {value}', 'es-ES': 'Editar {value}', 'fr-FR': 'Modifier {value}', 'de-DE': '{value} bearbeiten',
    'ja-JP': '{value}を編集', 'ko-KR': '{value} 편집', 'pt-PT': 'Editar {value}', 'ar-SA': 'تحرير {value}',
    'ru-RU': 'Изменить {value}', 'it-IT': 'Modifica {value}', 'zh-TW': '編輯{value}'
  },
  delete: {
    'en-US': 'Delete {value}', 'es-ES': 'Eliminar {value}', 'fr-FR': 'Supprimer {value}', 'de-DE': '{value} löschen',
    'ja-JP': '{value}を削除', 'ko-KR': '{value} 삭제', 'pt-PT': 'Eliminar {value}', 'ar-SA': 'حذف {value}',
    'ru-RU': 'Удалить {value}', 'it-IT': 'Elimina {value}', 'zh-TW': '刪除{value}'
  },
  required: {
    'en-US': '{value} is required', 'es-ES': '{value} es obligatorio', 'fr-FR': '{value} est obligatoire', 'de-DE': '{value} ist erforderlich',
    'ja-JP': '{value}は必須です', 'ko-KR': '{value} 필수', 'pt-PT': '{value} é obrigatório', 'ar-SA': '{value} مطلوب',
    'ru-RU': '{value} обязательно', 'it-IT': '{value} è obbligatorio', 'zh-TW': '{value}不能為空'
  },
  management: {
    'en-US': '{value} management', 'es-ES': 'Gestión de {value}', 'fr-FR': 'Gestion {value}', 'de-DE': '{value}-Verwaltung',
    'ja-JP': '{value}管理', 'ko-KR': '{value} 관리', 'pt-PT': 'Gestão de {value}', 'ar-SA': 'إدارة {value}',
    'ru-RU': 'Управление {value}', 'it-IT': 'Gestione {value}', 'zh-TW': '{value}管理'
  },
  detail: {
    'en-US': '{value} details', 'es-ES': 'Detalles de {value}', 'fr-FR': 'Détails {value}', 'de-DE': '{value}-Details',
    'ja-JP': '{value}詳細', 'ko-KR': '{value} 상세', 'pt-PT': 'Detalhes de {value}', 'ar-SA': 'تفاصيل {value}',
    'ru-RU': 'Сведения: {value}', 'it-IT': 'Dettagli {value}', 'zh-TW': '{value}詳情'
  },
  name: {
    'en-US': '{value} name', 'es-ES': 'Nombre de {value}', 'fr-FR': 'Nom {value}', 'de-DE': '{value}-Name',
    'ja-JP': '{value}名', 'ko-KR': '{value} 이름', 'pt-PT': 'Nome de {value}', 'ar-SA': 'اسم {value}',
    'ru-RU': 'Название: {value}', 'it-IT': 'Nome {value}', 'zh-TW': '{value}名稱'
  }
}

const generic = {
  'en-US': 'Item',
  'es-ES': 'Elemento',
  'fr-FR': 'Élément',
  'de-DE': 'Element',
  'ja-JP': '項目',
  'ko-KR': '항목',
  'pt-PT': 'Item',
  'ar-SA': 'عنصر',
  'ru-RU': 'Элемент',
  'it-IT': 'Elemento',
  'zh-TW': '項目'
}

const uiWords = {
  confirm: {
    'en-US': 'Confirm', 'es-ES': 'Confirmar', 'fr-FR': 'Confirmer', 'de-DE': 'Bestätigen',
    'ja-JP': '確認', 'ko-KR': '확인', 'pt-PT': 'Confirmar', 'ar-SA': 'تأكيد',
    'ru-RU': 'Подтвердить', 'it-IT': 'Conferma', 'zh-TW': '確認'
  },
  done: {
    'en-US': 'Done', 'es-ES': 'Listo', 'fr-FR': 'Terminé', 'de-DE': 'Erledigt',
    'ja-JP': '完了', 'ko-KR': '완료', 'pt-PT': 'Concluído', 'ar-SA': 'تم',
    'ru-RU': 'Готово', 'it-IT': 'Completato', 'zh-TW': '完成'
  },
  failed: {
    'en-US': 'Failed', 'es-ES': 'Error', 'fr-FR': 'Échec', 'de-DE': 'Fehler',
    'ja-JP': '失敗', 'ko-KR': '실패', 'pt-PT': 'Falhou', 'ar-SA': 'فشل',
    'ru-RU': 'Ошибка', 'it-IT': 'Errore', 'zh-TW': '失敗'
  },
  required: {
    'en-US': 'Required', 'es-ES': 'Obligatorio', 'fr-FR': 'Obligatoire', 'de-DE': 'Pflichtfeld',
    'ja-JP': '必須', 'ko-KR': '필수', 'pt-PT': 'Obrigatório', 'ar-SA': 'مطلوب',
    'ru-RU': 'Обязательно', 'it-IT': 'Obbligatorio', 'zh-TW': '必填'
  },
  select: {
    'en-US': 'Select', 'es-ES': 'Seleccionar', 'fr-FR': 'Sélectionner', 'de-DE': 'Auswählen',
    'ja-JP': '選択', 'ko-KR': '선택', 'pt-PT': 'Selecionar', 'ar-SA': 'اختر',
    'ru-RU': 'Выбрать', 'it-IT': 'Seleziona', 'zh-TW': '選擇'
  },
  enter: {
    'en-US': 'Enter', 'es-ES': 'Introducir', 'fr-FR': 'Saisir', 'de-DE': 'Eingeben',
    'ja-JP': '入力', 'ko-KR': '입력', 'pt-PT': 'Introduzir', 'ar-SA': 'أدخل',
    'ru-RU': 'Введите', 'it-IT': 'Inserisci', 'zh-TW': '輸入'
  },
  none: {
    'en-US': 'No data', 'es-ES': 'Sin datos', 'fr-FR': 'Aucune donnée', 'de-DE': 'Keine Daten',
    'ja-JP': 'データなし', 'ko-KR': '데이터 없음', 'pt-PT': 'Sem dados', 'ar-SA': 'لا توجد بيانات',
    'ru-RU': 'Нет данных', 'it-IT': 'Nessun dato', 'zh-TW': '暫無資料'
  },
  settings: {
    'en-US': 'Settings', 'es-ES': 'Ajustes', 'fr-FR': 'Réglages', 'de-DE': 'Einstellungen',
    'ja-JP': '設定', 'ko-KR': '설정', 'pt-PT': 'Definições', 'ar-SA': 'الإعدادات',
    'ru-RU': 'Настройки', 'it-IT': 'Impostazioni', 'zh-TW': '設定'
  },
  details: {
    'en-US': 'Details', 'es-ES': 'Detalles', 'fr-FR': 'Détails', 'de-DE': 'Details',
    'ja-JP': '詳細', 'ko-KR': '상세', 'pt-PT': 'Detalhes', 'ar-SA': 'التفاصيل',
    'ru-RU': 'Сведения', 'it-IT': 'Dettagli', 'zh-TW': '詳情'
  },
  note: {
    'en-US': 'Note', 'es-ES': 'Nota', 'fr-FR': 'Note', 'de-DE': 'Notiz',
    'ja-JP': 'メモ', 'ko-KR': '메모', 'pt-PT': 'Nota', 'ar-SA': 'ملاحظة',
    'ru-RU': 'Заметка', 'it-IT': 'Nota', 'zh-TW': '備註'
  }
}

const englishFallbackTerms = {
  '客户主面板': 'Workspace',
  '销售工作台': 'Sales workspace',
  '客服工作台': 'Service workspace',
  '管理者仪表盘': 'Manager dashboard',
  '销售优先事项': 'Sales priorities',
  '客服处理队列': 'Service queue',
  '经营概览': 'Business overview',
  '通知和待办': 'Notifications and tasks',
  '下一步建议': 'Next-step suggestions',
  '销售漏斗': 'Sales funnel',
  '服务压力': 'Service load',
  '工单总数': 'Total tickets',
  '超时未处理': 'Overdue',
  '已解决/关闭': 'Resolved / closed',
  '满意度': 'Satisfaction',
  '待处理线索': 'Pending leads',
  '进行中商机': 'Open opportunities',
  '订单履约': 'Order fulfillment',
  '服务工单': 'Service ticket',
  '线索': 'Lead',
  '商机': 'Opportunity',
  '订单': 'Order',
  '工单': 'Ticket',
  '市场活动': 'Campaign',
  '成交记录': 'Deal record',
  '客户项目': 'Customer project',
  '客户': 'Customer',
  '联系人': 'Contact',
  '公司': 'Company',
  '产品服务': 'Products and services',
  '团队管理': 'Team management',
  '账号权限': 'Account permissions',
  '运营热点': 'Operational insights',
  '销售大屏': 'Sales dashboard',
  '企业管理': 'Enterprise management',
  '系统': 'System',
  '平台': 'Platform',
  '工作区': 'Workspace',
  '关联金额': 'Related amount',
  '全部紧急程度': 'All priorities',
  '可登录账号': 'Login-enabled accounts',
  '分成说明': 'Commission notes',
  '请填写动态内容': 'Please enter activity content',
  '设备': 'Device',
  '完善输单复盘': 'Complete lost-deal review',
  '释放回待分配池': 'Release back to assignment pool',
  '进行中': 'In progress',
  '同客户': 'same customer',
  '重复跟进': 'duplicate follow-up',
  '成交/订单收入': 'Deal / order revenue',
  '培育': 'nurture',
  '套模板': 'apply template',
  '关键人物': 'Key person',
  '干跑检查': 'Dry-run check',
  '表单回填预览': 'Form fill preview',
  '已下线': 'Offline',
  '进入培育': 'enter nurture',
  '具体原因': 'specific reason',
  '老客户触达': 'existing-customer outreach',
  '文化娱乐': 'Media and entertainment',
  '租户ID': 'Tenant ID',
  '清理旧消息': 'Clean old messages',
  '下移': 'Move down',
  'AI余额': 'AI balance',
  '手动分配': 'Manual assignment',
  '客户360': 'Customer 360',
  '线索分配成功': 'Lead assigned successfully',
  '交付项目': 'Delivery project',
  '未开始': 'Not started',
  '阻塞/风险': 'Blocked / risk',
  '无需继续处理': 'No further action needed',
  '成员数': 'Members',
  '触点归因': 'Touch attribution',
  '间隔分钟': 'Interval minutes',
  '自动化规则': 'automation rules',
  '折算': 'Converted',
  '制度公告': 'Policy notice',
  '宣传页': 'Brochure',
  '目标公司名': 'target company name',
  '重点客户': 'key accounts',
  '订单数': 'Orders',
  '付款计划': 'payment plan',
  '订单资料': 'Order files',
  '权限口径': 'Permission scope',
  '活动背景': 'Campaign background',
  '跟进话术': 'follow-up script',
  '目标客户说明': 'target customer notes',
  '预估金额': 'Estimated amount',
  '当前进度': 'Current progress',
  '拒绝入培育': 'Reject nurture',
  '编码': 'Code',
  '公司规则': 'company rules',
  '操作按钮': 'action buttons',
  '不限': 'Unlimited',
  '操作失败': 'Operation failed',
  '洞察': 'Insights',
  '当前阶段': 'Current stage',
  '物流单号': 'tracking number',
  '未收款': 'unpaid',
  '跟进': 'Follow-up',
  '醫療健康': 'Healthcare',
  '医疗健康': 'Healthcare',
  'UTM参数': 'UTM parameters',
  '总部/地区': 'Headquarters / region',
  '关联商机': 'Related opportunity',
  '归档/异常': 'Archived / exceptions',
  '活动类型': 'Campaign type',
  '电话邀约': 'Phone invitation',
  '待补齐': 'To be completed',
  '互动类型': 'Interaction type',
  '活动动态已记录': 'Campaign activity recorded',
  '库存': 'Inventory',
  '订单明细': 'order items',
  '更新项目进度': 'Update project progress',
  '交通運輸': 'Transportation',
  '交通运输': 'Transportation',
  '新建': 'New',
  '待办任务': 'task',
  '手工录入': 'Manual entry',
  '计划完成': 'Planned completion',
  '库存预警线': 'Inventory warning threshold',
  '文件资料': 'Files',
  '发票/免票': 'Invoice / exemption',
  '商业模式': 'Business model',
  '预算': 'Budget',
  '仅未培育': 'Only not nurtured',
  '系统识别': 'System detected',
  '活动线索': 'Campaign leads',
  '评分低于': 'Score below',
  '订单资料已上传': 'Order file uploaded',
  '客户健康度': 'Customer health',
  '草稿': 'Draft',
  '库存金额': 'Inventory value',
  '岗位编码': 'Post code',
  '方案': 'Proposal',
  '未设置岗位': 'No post set',
  '金融服务': 'Financial services',
  '来源追踪参数': 'Source tracking parameters',
  '跨境電商': 'Cross-border ecommerce',
  '跨境电商': 'Cross-border ecommerce',
  '付款条件': 'Payment terms',
  '评分依据': 'scoring basis',
  '访问权限': 'access permissions',
  '组织部门': 'Organization departments',
  '不修复': 'Do not fix',
  '手机或邮箱': 'mobile or email',
  '只读': 'read-only',
  '发送': 'Send',
  '合同': 'contract',
  '开票': 'invoicing',
  '复购': 'repeat purchase',
  '截图': 'screenshots',
  '日志': 'logs',
  '客户反馈': 'customer feedback',
  '处理说明': 'handling notes',
  '普通用户怎么填': 'How should regular users fill this in',
  '已赢单': 'Won',
  '培育中': 'Nurturing',
  '状态变更': 'Status change',
  '生物医药': 'Biopharma',
  '已过期': 'Expired',
  '线索状态已更新': 'Lead status updated',
  '跟进确认客户兴趣': 'Follow up to confirm customer interest',
  '工单已保存': 'Ticket saved',
  '变更订单状态': 'Change order status',
  '热点客户': 'Hot customers',
  '目标线索数': 'Target leads',
  '切入话术': 'Opening script',
  '线索转成商机': 'Convert lead to opportunity',
  '线索评分': 'Lead scoring',
  '再营销广告': 'remarketing ads',
  '平台能力权限': 'Platform capability permissions',
  '批量导入': 'Batch import',
  '跟进协作': 'Follow-up collaboration',
  '关联联系人': 'Related contact',
  '可处理线索': 'actionable leads',
  '交易风险': 'Deal risk',
  '跟进任务': 'Follow-up tasks',
  '发布全员通知': 'Publish companywide notice',
  '重点内容': 'Key content',
  '内容资料': 'Content files',
  '当前用户': 'Current user',
  '需要关注': 'Needs attention',
  '加载产品失败': 'Failed to load products',
  '产品数量': 'Product quantity',
  '保存失败': 'Save failed',
  '已有联系人': 'existing contacts',
  '处理时限': 'Handling deadline',
  '来源活动': 'source campaigns',
  '刚收到问题': 'New issue received',
  '当前活动': 'Current campaign',
  '升级队列': 'escalation queue',
  '大屏模式': 'Dashboard mode',
  '权限标识': 'Permission key',
  '未开通': 'Not enabled',
  '线索自动化规则': 'Lead automation rules',
  '提醒标题': 'Reminder title',
  '维护': 'Maintain',
  '基础资料': 'basic profile',
  '联系信息': 'contact information',
  '活动即将结束': 'Campaign ending soon',
  '自动分派工单': 'Ticket auto-assigned',
  '私密项目': 'Private project',
  '路由': 'Route',
  '操作类型': 'Operation type',
  '在售': 'On sale',
  '其他': 'Other',
  '会议纪要': 'Meeting notes',
  '全部状态': 'All statuses',
  '知识推荐': 'Knowledge recommendations',
  '配置权限': 'Configure permissions',
  '明确需求': 'clear need',
  '报价': 'quote',
  '知识库': 'knowledge base',
  '客户名称': 'customer name',
  '协作权限记录': 'collaboration permission record',
  '自然社交': 'Organic social',
  '平均客单价': 'Average order value',
  '已确认/待履行': 'Confirmed / pending fulfillment',
  '未找到关联商机': 'Related opportunity not found',
  '上传项目资料': 'Upload project files',
  '名称': 'Name',
  '邮件': 'Email',
  '通知类型': 'Notification type',
  '附件资料': 'Attachments',
  '最近动态': 'Recent activity',
  '日志审计': 'Audit log',
  '已拒绝转入培育': 'Rejected nurture transfer',
  '选择协作人': 'Select collaborator',
  '未读': 'Unread',
  '内嵌': 'Embedded',
  '落地页': 'Landing page',
  '批量导入线索': 'Batch import leads',
  '图标': 'Icon',
  '开始日期': 'Start date',
  '线上研讨会': 'Webinar',
  '素材上传与查看': 'Upload and view materials',
  '订单明细': 'Order items',
  '请输入': 'Please enter',
  '请选择': 'Please select',
  '暂无': 'No',
  '全部': 'All',
  '当前': 'Current',
  '关联': 'Related',
  '金额': 'Amount',
  '紧急程度': 'Priority',
  '登录': 'Login',
  '账号': 'Account',
  '说明': 'Notes',
  '填写': 'Enter',
  '动态': 'Activity',
  '内容': 'Details',
  '时间': 'Time',
  '日期': 'Date',
  '状态': 'Status',
  '类型': 'Type',
  '阶段': 'Stage',
  '来源': 'Source',
  '渠道': 'Channel',
  '活动': 'Campaign',
  '归因': 'Attribution',
  '收入': 'Revenue',
  '成本': 'Cost',
  '目标': 'Target',
  '成员': 'Member',
  '用户': 'User',
  '角色': 'Role',
  '部门': 'Department',
  '岗位': 'Post',
  '菜单': 'Menu',
  '权限': 'Permission',
  '数据': 'Data',
  '字段': 'Field',
  '规则': 'Rule',
  '模板': 'Template',
  '任务': 'Task',
  '待办': 'Task',
  '通知': 'Notification',
  '消息': 'Message',
  '提醒': 'Reminder',
  '处理': 'Handle',
  '完成': 'Complete',
  '保存': 'Save',
  '删除': 'Delete',
  '编辑': 'Edit',
  '新增': 'Create',
  '创建': 'Create',
  '生成': 'Generate',
  '查看': 'View',
  '查询': 'Search',
  '搜索': 'Search',
  '刷新': 'Refresh',
  '重置': 'Reset',
  '导入': 'Import',
  '导出': 'Export',
  '上传': 'Upload',
  '下载': 'Download',
  '拒绝': 'Reject',
  '释放': 'Release',
  '分配': 'Assign',
  '协作': 'Collaborate',
  '公开': 'Public',
  '私密': 'Private',
  '赢单': 'Won',
  '输单': 'Lost',
  '搁置': 'On hold',
  '初步接触': 'Initial contact',
  '需求确认': 'Needs confirmed',
  '方案/报价': 'Proposal / quote',
  '谈判/评审': 'Negotiation / review',
  '待处理': 'Pending',
  '跟进中': 'Following',
  '已分派': 'Assigned',
  '处理中': 'Processing',
  '已解决': 'Resolved',
  '已关闭': 'Closed',
  '普通': 'Normal',
  '紧急': 'Urgent',
  '高优先级': 'High priority',
  '低优先级': 'Low priority',
  '高': 'High',
  '中': 'Medium',
  '低': 'Low',
  '人': 'people',
  '以上': 'or more',
  '以下': 'or less',
  '以内': 'within',
  '多少': 'how many',
  '个': '',
  '条': '',
  '套': 'sets',
  '万': 'ten thousand',
  '确定删除': 'Confirm deletion',
  '删除后不可恢复': 'This cannot be undone after deletion',
  '不可恢复': 'cannot be undone',
  '确认发布': 'Confirm publish',
  '确认将当前收件箱中的系统消息全部标记为已读吗': 'Mark all system messages in the current inbox as read',
  '系统消息已归档': 'System message archived',
  '公司名、行业、官网、等级': 'Company name, industry, website, and grade',
  '成交、客户、合同号': 'Deal, customer, or contract number',
  '订单是成交后的履约单': 'Orders are fulfillment records after a deal is closed',
  '先确认订单': 'Confirm the order first',
  '登记发货': 'record shipment',
  '取消订单': 'cancel an order',
  '业务模块权限': 'Business module permissions',
  '菜单显示': 'menu visibility',
  '动作控制': 'action control',
  '平台能力': 'Platform capabilities',
  '模块权限速查': 'Module permission quick reference',
  '访问范围': 'Access scope',
  '公开数据': 'public data',
  '私密数据': 'private data',
  '负责人': 'owner',
  '创建人': 'creator',
  '指定协作人': 'specified collaborators',
  '活动还未开始': 'Campaign has not started',
  '活动未保存': 'Campaign is not saved',
  '无法上传素材': 'materials cannot be uploaded',
  '上传素材': 'Upload materials',
  '广告素材': 'Ad materials',
  '报价资料': 'Quote files',
  '验收资料': 'Acceptance files',
  '选择文件': 'Select file',
  '未知大小': 'Unknown size',
  '税码': 'Tax code',
  '税率': 'Tax rate',
  '本位币金额': 'Base currency amount',
  '已回款': 'Paid amount',
  '首付款': 'Down payment',
  '收齐回款': 'Collect full payment',
  '发票或免票原因': 'Invoice or exemption reason',
  '发票类型': 'Invoice type',
  '输单原因': 'Lost reason',
  '交易链路': 'Deal chain',
  '上下游关系': 'upstream and downstream relationships',
  '自动补全': 'automatically complete',
  '当前商机还没有生成报价、成交记录或订单': 'This opportunity has no quote, deal record, or order yet',
  '商机画像': 'Opportunity profile',
  '商机阶段已推进': 'Opportunity stage advanced',
  '商机还没有形成订单': 'Opportunity has not created an order yet',
  '补充产品后才能生成更准确的报价': 'Add products to generate a more accurate quote',
  '客户接受报价': 'Customer accepted quote',
  '发送报价单': 'Send quote',
  '成交处理': 'Deal handling',
  '已生成订单草稿': 'Order draft created',
  '可到订单页面继续处理': 'continue processing on the order page',
  'SLA超时': 'SLA overdue',
  'SLA升级队列': 'SLA escalation queue',
  '手动升级': 'manual escalation',
  '处理情况': 'Handling status',
  '复现描述': 'Reproduction description',
  '复现步骤': 'Reproduction steps',
  '报错信息': 'error details',
  '截图说明': 'screenshot notes',
  '网络安全': 'Cybersecurity',
  '食品飲料': 'Food and beverage',
  '電子商務': 'Ecommerce',
  '顧問服務': 'Consulting services',
  '選擇或輸入行業': 'Select or enter industry',
  '選擇或輸入公司規模': 'Select or enter company size',
  '座機、總機或公司聯絡電話': 'Landline, switchboard, or company phone',
  '潜在': 'Potential',
  '潛在': 'Potential',
  '直接访问': 'Direct access',
  '其他获客来源': 'Other acquisition source',
  'CRM触达': 'CRM outreach',
  '运营触达': 'Operations outreach',
  '站内提醒': 'In-app reminder',
  '站内提醒 + 跟进任务': 'In-app reminder + follow-up task',
  '活动/会议': 'Campaign / meeting',
  '线下活动/展会': 'Offline event / expo',
  '展会/线下活动': 'Expo / offline event',
  '展会、会议、沙龙': 'Expo, meeting, or salon',
  '社交工具': 'Social tools',
  '社交消息': 'Social message',
  '广告投放': 'Advertising',
  '更多来源设置': 'More source settings',
  '效果复盘': 'Performance review',
  '高级效果复盘': 'Advanced performance review',
  '活动带来的业务结果': 'Business results from the campaign',
  '公司背景': 'Company background',
  '主营业务': 'main business',
  '销售切入点': 'sales entry points',
  '风险提示': 'risk notes',
  '主营产品/服务': 'Main products / services',
  '线索漏斗': 'Lead funnel',
  '今天新增线索': 'New leads today',
  '活动线索还没有转成商机': 'Campaign leads have not been converted to opportunities',
  '已转商机': 'Converted to opportunity',
  '疑似重复记录': 'possible duplicate records',
  '重复提交': 'duplicate submission',
  '同一次获客': 'same acquisition event',
  '同一业务意向': 'same business intent',
  '可以继续创建新线索': 'You can continue creating a new lead',
  '同一公司允许存在多条不同需求线索': 'The same company can have multiple leads for different needs',
  '联系人待补齐': 'Contact to be completed',
  '尚未指定负责人': 'Owner not assigned',
  '等主管确认': 'Waiting for manager confirmation',
  'SDR主管': 'SDR manager',
  '主管 / SDR主管': 'Manager / SDR manager',
  'SDR确认有明确商机': 'SDR confirmed a clear opportunity',
  '已标记为SQL': 'Marked as SQL',
  '已分配待跟进': 'Assigned and waiting for follow-up',
  '已自动分派工单': 'Ticket automatically assigned',
  '已处理': 'Handled',
  '已阻止': 'Blocked',
  '已结束': 'Ended',
  '开启': 'Enabled',
  '恢复启用': 'Re-enable',
  '调度': 'Schedule',
  '下次执行': 'Next run',
  '冷却小时': 'Cooldown hours',
  '保存快照': 'Save snapshot',
  '配置销售目标': 'configure sales targets',
  '销售目标': 'sales targets',
  '赢单金额走势': 'won amount trend',
  '增长有没有掉速': 'whether growth is slowing',
  '健康状态': 'Health status',
  '健康': 'Healthy',
  '有风险': 'At risk',
  '高价值': 'High value',
  '业务提醒': 'Business reminder',
  '目标客户公司名': 'target customer company name',
  '每行一个目标客户公司名': 'One target customer company per line',
  '每行一个目标公司名': 'One target company name per line',
  '隐藏菜单': 'Hide menu',
  '上级菜单': 'Parent menu',
  '顶级菜单': 'top-level menu',
  '昵称': 'Nickname',
  '当前密码': 'Current password',
  '新密码至少需要8位': 'New password must be at least 8 characters',
  '两次输入密码不一致': 'The two passwords do not match',
  '更新密码': 'Update password',
  '退出其他会话': 'Sign out other sessions',
  '访问时间': 'Access time',
  '发送方': 'Sender',
  '系统基础模板不可删除': 'System base templates cannot be deleted',
  '基础模板': 'Base template',
  '保存模板': 'Save template',
  '自定义模板': 'custom template',
  '模板不可删除': 'template cannot be deleted',
  '什么时候用': 'When to use',
  '没有匹配的术语': 'No matching terms',
  '直接改为': 'Change directly to',
  '该资料': 'This file',
  '下次': 'Next',
  '分钟': 'minutes',
  '女': 'Female'

}

export function hasHanText(value = '') {
  return /[\u3400-\u9FFF]/.test(String(value || ''))
}

export function toTraditionalText(value = '') {
  return String(value ?? '')
    .replaceAll('线索', '線索')
    .replaceAll('联系人', '聯絡人')
    .replaceAll('客户', '客戶')
    .replaceAll('业务', '業務')
    .replaceAll('服务', '服務')
    .replaceAll('发送', '傳送')
    .replaceAll('统一', '統一')
    .replaceAll('访问', '存取')
    .replaceAll('继承', '繼承')
    .replaceAll('对应', '對應')
    .replaceAll('真实', '真實')
    .replaceAll('变化', '變化')
    .replaceAll('评分', '評分')
    .replaceAll('作废', '作廢')
    .replaceAll('截图', '截圖')
    .replaceAll('反馈', '回饋')
    .replaceAll('行业', '產業')
    .replaceAll('官网', '官網')
    .replaceAll('等级', '等級')
    .replaceAll('关系', '關係')
    .replaceAll('销售', '銷售')
    .replaceAll('复杂', '複雜')
    .replaceAll('币种', '幣種')
    .replaceAll('汇率', '匯率')
    .replaceAll('税费', '稅費')
    .replaceAll('计算', '計算')
    .replaceAll('运营', '營運')
    .replaceAll('触达', '觸達')
    .replaceAll('供应链', '供應鏈')
    .replaceAll('团队', '團隊')
    .replaceAll('部门', '部門')
    .replaceAll('目标', '目標')
    .replaceAll('情况', '情況')
    .replaceAll('新增', '新增')
    .replaceAll('平台', '平台')
    .replaceAll('能力', '能力')
    .replaceAll('给', '給')
    .replaceAll('主管', '主管')
    .replaceAll('热点', '熱點')
    .replaceAll('系统', '系統')
    .replaceAll('异常', '異常')
    .replaceAll('机会', '機會')
    .replaceAll('现在', '現在')
    .replaceAll('值得', '值得')
    .replaceAll('处理', '處理')
    .replaceAll('什么', '什麼')
    .replaceAll('报表', '報表')
    .replaceAll('大屏', '大屏')
    .replaceAll('告诉', '告訴')
    .replaceAll('集中', '集中')
    .replaceAll('目标客户', '目標客戶')
    .replaceAll('组织', '組織')
    .replaceAll('账号', '帳號')
    .replaceAll('用户', '使用者')
    .replaceAll('用户名', '使用者名稱')
    .replaceAll('动作', '動作')
    .replaceAll('口径', '口徑')
    .replaceAll('单号', '單號')
    .replaceAll('单', '單')
    .replaceAll('站内', '站內')
    .replaceAll('物流', '物流')
    .replaceAll('售后', '售後')
    .replaceAll('模块', '模組')
    .replaceAll('页面', '頁面')
    .replaceAll('说明', '說明')
    .replaceAll('视频', '影片')
    .replaceAll('资料', '資料')
    .replaceAll('报告', '報告')
    .replaceAll('指南', '指南')
    .replaceAll('线上', '線上')
    .replaceAll('直播', '直播')
    .replaceAll('会议', '會議')
    .replaceAll('邮件', '郵件')
    .replaceAll('广告', '廣告')
    .replaceAll('字段', '欄位')
    .replaceAll('统计', '統計')
    .replaceAll('归集', '歸集')
    .replaceAll('归档', '歸檔')
    .replaceAll('来源', '來源')
    .replaceAll('宣传', '宣傳')
    .replaceAll('白皮书', '白皮書')
    .replaceAll('海报', '海報')
    .replaceAll('价值', '價值')
    .replaceAll('报名', '報名')
    .replaceAll('说明会', '說明會')
    .replaceAll('当前', '目前')
    .replaceAll('活动', '活動')
    .replaceAll('手动', '手動')
    .replaceAll('自动', '自動')
    .replaceAll('规则', '規則')
    .replaceAll('阶段', '階段')
    .replaceAll('维护', '維護')
    .replaceAll('产品', '產品')
    .replaceAll('明细', '明細')
    .replaceAll('建议', '建議')
    .replaceAll('编码', '編碼')
    .replaceAll('勾选', '勾選')
    .replaceAll('权限', '權限')
    .replaceAll('自定义', '自訂')
    .replaceAll('按钮', '按鈕')
    .replaceAll('只读', '唯讀')
    .replaceAll('进入', '進入')
    .replaceAll('点击', '點擊')
    .replaceAll('这里', '這裡')
    .replaceAll('该', '該')
    .replaceAll('后', '後')
    .replaceAll('会', '會')
    .replaceAll('条', '條')
    .replaceAll('与', '與')
    .replaceAll('并', '並')
    .replaceAll('对接', '對接')
    .replaceAll('填写', '填寫')
    .replaceAll('清楚', '清楚')
    .replaceAll('原因', '原因')
    .replaceAll('取消', '取消')
    .replaceAll('确认', '確認')
    .replaceAll('开票', '開票')
    .replaceAll('登记', '登記')
    .replaceAll('发货', '發貨')
    .replaceAll('项目', '專案')
    .replaceAll('订单', '訂單')
    .replaceAll('商机', '商機')
    .replaceAll('记录', '記錄')
    .replaceAll('链路', '鏈路')
    .replaceAll('对象', '物件')
    .replaceAll('数据', '資料')
    .replaceAll('消息', '訊息')
    .replaceAll('文件', '檔案')
    .replaceAll('节点', '節點')
    .replaceAll('上下游', '上下游')
    .replaceAll('导出', '匯出')
    .replaceAll('导入', '匯入')
    .replaceAll('设置', '設定')
    .replaceAll('查询', '查詢')
    .replaceAll('查看', '查看')
    .replaceAll('状态', '狀態')
    .replaceAll('创建', '建立')
    .replaceAll('生成', '產生')
    .replaceAll('更新', '更新')
    .replaceAll('删除', '刪除')
    .replaceAll('启用', '啟用')
    .replaceAll('禁用', '停用')
    .replaceAll('登录', '登入')
    .replaceAll('密码', '密碼')
    .replaceAll('验证码', '驗證碼')
    .replaceAll('验证', '驗證')
    .replaceAll('激活', '啟用')
    .replaceAll('失败', '失敗')
    .replaceAll('检查', '檢查')
    .replaceAll('链接', '連結')
    .replaceAll('过期', '過期')
    .replaceAll('暂无', '暫無')
    .replaceAll('请输入', '請輸入')
    .replaceAll('请选择', '請選擇')
    .replaceAll('请', '請')
    .replaceAll('选择', '選擇')
    .replaceAll('识别', '識別')
    .replaceAll('继续', '繼續')
    .replaceAll('允许', '允許')
    .replaceAll('优先', '優先')
    .replaceAll('输入', '輸入')
    .replaceAll('协作', '協作')
    .replaceAll('报价', '報價')
    .replaceAll('回款', '回款')
    .replaceAll('履约', '履約')
    .replaceAll('赢单', '贏單')
    .replaceAll('负责人', '負責人')
    .replaceAll('电话', '電話')
    .replaceAll('邮箱', '信箱')
    .replaceAll('数量', '數量')
    .replaceAll('金额', '金額')
    .replaceAll('页', '頁')
    .replaceAll('后', '後')
    .replaceAll('无', '無')
    .replaceAll('复', '復')
}

export function isStrictLocaleValueUsable(value, locale) {
  if (value === undefined || value === null) return false
  const text = String(value).trim()
  if (!text || /^domain\.auto\.[a-f0-9]+$/i.test(text)) return false
  if (nonHanLocales.has(locale) && hasHanText(text)) return false
  if (locale === 'ko-KR' && hasHanText(text)) return false
  return true
}

export function localizeGeneratedText(source, locale, key = '') {
  const text = String(source ?? '').trim()
  if (!text) return strictTranslationMissingByLocale[locale] || strictTranslationMissingByLocale['zh-CN']
  if (looksLikeCodeFragment(text)) return compactFallbackFromKey(key, locale)
  if (locale === 'zh-CN') return text
  if (locale === 'zh-TW') return exactByLocale['zh-TW'][text] || toTraditionalText(text)
  const exactText = exactByLocale[locale]?.[text]
  if (exactText) return exactText

  const pattern = localizeByPattern(text, locale)
  if (pattern) return pattern

  const replaced = replaceKnownTerms(text, locale)
  if (isStrictLocaleValueUsable(replaced, locale) && replaced !== text) return replaced

  const sentence = localizeBySentence(text, locale)
  if (sentence) return sentence

  const fallback = localizeByFallbackTerms(text, locale)
  if (fallback) return fallback

  return humanizeFallbackKey(key, locale)
}

function looksLikeCodeFragment(text) {
  return /&&|<\/?|v-model|pagination|Icon icon|completionWarnings|class=|plain>|{{|}}|\\+$/.test(String(text || ''))
}

function localizeByPattern(text, locale) {
  const patterns = [
    [/^请输入(.+)$/, 'enter'],
    [/^请选择(.+)$/, 'select'],
    [/^暂无(.+)$/, 'noData'],
    [/^新增(.+)$/, 'create'],
    [/^编辑(.+)$/, 'edit'],
    [/^删除(.+)$/, 'delete'],
    [/^(.+)不能为空$/, 'required'],
    [/^(.+)管理$/, 'management'],
    [/^(.+)详情$/, 'detail'],
    [/^(.+)名称$/, 'name']
  ]
  for (const [re, templateKey] of patterns) {
    const match = text.match(re)
    if (!match) continue
    const value = localizeObjectText(match[1], locale)
    return fillTemplate(templates[templateKey]?.[locale], value)
  }
  return ''
}

function localizeObjectText(text, locale) {
  const normalized = String(text || '').trim()
  if (!normalized) return generic[locale] || ''
  const exactText = exactByLocale[locale]?.[normalized]
  if (exactText) return exactText
  const replaced = replaceKnownTerms(normalized, locale)
  if (isStrictLocaleValueUsable(replaced, locale) && replaced !== normalized) return replaced
  return compactFallbackFromText(normalized, locale)
}

function fillTemplate(template, value) {
  if (!template) return ''
  return template.replaceAll('{value}', value)
}

function replaceKnownTerms(text, locale) {
  let output = String(text || '')
  const entries = Object.entries(exactByLocale[locale] || {}).sort((a, b) => b[0].length - a[0].length)
  for (const [zh, localized] of entries) {
    output = output.replaceAll(zh, localized)
  }
  return output
}

function localizeBySentence(text, locale) {
  const source = String(text || '').trim()
  if (!source || locale === 'zh-CN') return ''
  const rules = [
    [/删除.*不可恢复|不可恢复|确认删除|确定删除/, 'Confirm delete'],
    [/确定|确认|是否|吗？|吗$/, 'Confirm'],
    [/不能为空|至少填写|至少需要/, 'Required'],
    [/请输入|请填写|请简要/, 'Enter'],
    [/请选择|选择文件|选择产品|选择协作人/, 'Select'],
    [/暂无|还没有|尚未|没有匹配|未找到|未设置|未填写|待补齐/, 'No data'],
    [/失败|加载.*失败|保存.*失败|更新.*失败|归档失败|创建失败/, 'Failed'],
    [/成功|已保存|已上传|已完成|已导出|已记录|已生成|已归档|已处理|已开通|已分派|已标记|已推进|已调整/, 'Done']
  ]
  for (const [re, enText] of rules) {
    if (!re.test(source)) continue
    return localizeCompactEnglish(enText, locale)
  }
  return ''
}

function localizeCompactEnglish(enText, locale) {
  if (locale === 'en-US') return enText
  const map = compactEnglishByLocale[enText]
  return map?.[locale] || enText
}

const compactEnglishByLocale = {
  'Confirm delete': {
    'es-ES': 'Confirmar borrado', 'fr-FR': 'Confirmer la suppression', 'de-DE': 'Löschen bestätigen',
    'ja-JP': '削除を確認', 'ko-KR': '삭제 확인', 'pt-PT': 'Confirmar eliminação', 'ar-SA': 'تأكيد الحذف',
    'ru-RU': 'Подтвердить удаление', 'it-IT': 'Conferma eliminazione', 'zh-TW': '確認刪除'
  },
  'Confirm': uiWords.confirm,
  'Required': uiWords.required,
  'Enter': uiWords.enter,
  'Select': uiWords.select,
  'Enter details': {
    'es-ES': 'Introducir datos', 'fr-FR': 'Saisir les détails', 'de-DE': 'Details eingeben',
    'ja-JP': '詳細を入力', 'ko-KR': '상세 입력', 'pt-PT': 'Introduzir detalhes', 'ar-SA': 'أدخل التفاصيل',
    'ru-RU': 'Введите данные', 'it-IT': 'Inserisci dettagli', 'zh-TW': '輸入詳情'
  },
  'Select option': {
    'es-ES': 'Seleccionar opción', 'fr-FR': 'Sélectionner une option', 'de-DE': 'Option wählen',
    'ja-JP': '選択してください', 'ko-KR': '옵션 선택', 'pt-PT': 'Selecionar opção', 'ar-SA': 'اختر خيارًا',
    'ru-RU': 'Выберите вариант', 'it-IT': 'Seleziona opzione', 'zh-TW': '選擇選項'
  },
  'No data': uiWords.none,
  'Failed': uiWords.failed,
  'Done': uiWords.done,
  'Access': {
    'es-ES': 'Acceso', 'fr-FR': 'Accès', 'de-DE': 'Zugriff',
    'ja-JP': 'アクセス', 'ko-KR': '접근', 'pt-PT': 'Acesso', 'ar-SA': 'الوصول',
    'ru-RU': 'Доступ', 'it-IT': 'Accesso', 'zh-TW': '存取'
  },
  'Notice': {
    'es-ES': 'Aviso', 'fr-FR': 'Avis', 'de-DE': 'Hinweis',
    'ja-JP': '通知', 'ko-KR': '알림', 'pt-PT': 'Aviso', 'ar-SA': 'إشعار',
    'ru-RU': 'Уведомление', 'it-IT': 'Avviso', 'zh-TW': '通知'
  },
  'Lead': {
    'es-ES': 'Lead', 'fr-FR': 'Prospect', 'de-DE': 'Lead',
    'ja-JP': 'リード', 'ko-KR': '리드', 'pt-PT': 'Lead', 'ar-SA': 'عميل محتمل',
    'ru-RU': 'Лид', 'it-IT': 'Lead', 'zh-TW': '線索'
  },
  'Deal': {
    'es-ES': 'Trato', 'fr-FR': 'Affaire', 'de-DE': 'Deal',
    'ja-JP': '商談', 'ko-KR': '거래', 'pt-PT': 'Negócio', 'ar-SA': 'صفقة',
    'ru-RU': 'Сделка', 'it-IT': 'Accordo', 'zh-TW': '交易'
  },
  'Order': {
    'es-ES': 'Pedido', 'fr-FR': 'Commande', 'de-DE': 'Auftrag',
    'ja-JP': '注文', 'ko-KR': '주문', 'pt-PT': 'Encomenda', 'ar-SA': 'طلب',
    'ru-RU': 'Заказ', 'it-IT': 'Ordine', 'zh-TW': '訂單'
  },
  'Campaign': {
    'es-ES': 'Campaña', 'fr-FR': 'Campagne', 'de-DE': 'Kampagne',
    'ja-JP': 'キャンペーン', 'ko-KR': '캠페인', 'pt-PT': 'Campanha', 'ar-SA': 'حملة',
    'ru-RU': 'Кампания', 'it-IT': 'Campagna', 'zh-TW': '活動'
  },
  'Customer': {
    'es-ES': 'Cliente', 'fr-FR': 'Client', 'de-DE': 'Kunde',
    'ja-JP': '顧客', 'ko-KR': '고객', 'pt-PT': 'Cliente', 'ar-SA': 'عميل',
    'ru-RU': 'Клиент', 'it-IT': 'Cliente', 'zh-TW': '客戶'
  },
  'Project': {
    'es-ES': 'Proyecto', 'fr-FR': 'Projet', 'de-DE': 'Projekt',
    'ja-JP': 'プロジェクト', 'ko-KR': '프로젝트', 'pt-PT': 'Projeto', 'ar-SA': 'مشروع',
    'ru-RU': 'Проект', 'it-IT': 'Progetto', 'zh-TW': '專案'
  },
  'Ticket': {
    'es-ES': 'Ticket', 'fr-FR': 'Ticket', 'de-DE': 'Ticket',
    'ja-JP': 'チケット', 'ko-KR': '티켓', 'pt-PT': 'Ticket', 'ar-SA': 'تذكرة',
    'ru-RU': 'Заявка', 'it-IT': 'Ticket', 'zh-TW': '工單'
  },
  'Product': {
    'es-ES': 'Producto', 'fr-FR': 'Produit', 'de-DE': 'Produkt',
    'ja-JP': '製品', 'ko-KR': '제품', 'pt-PT': 'Produto', 'ar-SA': 'منتج',
    'ru-RU': 'Продукт', 'it-IT': 'Prodotto', 'zh-TW': '產品'
  },
  'Admin': {
    'es-ES': 'Admin', 'fr-FR': 'Admin', 'de-DE': 'Admin',
    'ja-JP': '管理', 'ko-KR': '관리', 'pt-PT': 'Admin', 'ar-SA': 'إدارة',
    'ru-RU': 'Админ', 'it-IT': 'Admin', 'zh-TW': '管理'
  },
  'Service': {
    'es-ES': 'Servicio', 'fr-FR': 'Service', 'de-DE': 'Service',
    'ja-JP': 'サービス', 'ko-KR': '서비스', 'pt-PT': 'Serviço', 'ar-SA': 'خدمة',
    'ru-RU': 'Сервис', 'it-IT': 'Servizio', 'zh-TW': '服務'
  },
  'Field': {
    'es-ES': 'Campo', 'fr-FR': 'Champ', 'de-DE': 'Feld',
    'ja-JP': '項目', 'ko-KR': '필드', 'pt-PT': 'Campo', 'ar-SA': 'حقل',
    'ru-RU': 'Поле', 'it-IT': 'Campo', 'zh-TW': '欄位'
  },
  'Access settings': {
    'es-ES': 'Acceso', 'fr-FR': 'Accès', 'de-DE': 'Zugriff',
    'ja-JP': 'アクセス', 'ko-KR': '접근 권한', 'pt-PT': 'Acesso', 'ar-SA': 'الوصول',
    'ru-RU': 'Доступ', 'it-IT': 'Accesso', 'zh-TW': '存取設定'
  },
  'Notifications': {
    'es-ES': 'Notificaciones', 'fr-FR': 'Notifications', 'de-DE': 'Benachrichtigungen',
    'ja-JP': '通知', 'ko-KR': '알림', 'pt-PT': 'Notificações', 'ar-SA': 'الإشعارات',
    'ru-RU': 'Уведомления', 'it-IT': 'Notifiche', 'zh-TW': '通知'
  },
  'Ticket handling': {
    'es-ES': 'Tickets', 'fr-FR': 'Tickets', 'de-DE': 'Tickets',
    'ja-JP': 'チケット対応', 'ko-KR': '티켓 처리', 'pt-PT': 'Tickets', 'ar-SA': 'التذاكر',
    'ru-RU': 'Заявки', 'it-IT': 'Ticket', 'zh-TW': '工單處理'
  },
  'Files': {
    'es-ES': 'Archivos', 'fr-FR': 'Fichiers', 'de-DE': 'Dateien',
    'ja-JP': 'ファイル', 'ko-KR': '파일', 'pt-PT': 'Ficheiros', 'ar-SA': 'الملفات',
    'ru-RU': 'Файлы', 'it-IT': 'File', 'zh-TW': '檔案'
  },
  'Order finance': {
    'es-ES': 'Pedidos', 'fr-FR': 'Commandes', 'de-DE': 'Aufträge',
    'ja-JP': '注文財務', 'ko-KR': '주문 재무', 'pt-PT': 'Encomendas', 'ar-SA': 'الطلبات',
    'ru-RU': 'Заказы', 'it-IT': 'Ordini', 'zh-TW': '訂單財務'
  },
  'Deal progress': {
    'es-ES': 'Avance del trato', 'fr-FR': 'Avancement', 'de-DE': 'Deal-Fortschritt',
    'ja-JP': '案件進捗', 'ko-KR': '거래 진행', 'pt-PT': 'Progresso', 'ar-SA': 'تقدم الصفقة',
    'ru-RU': 'Ход сделки', 'it-IT': 'Avanzamento', 'zh-TW': '交易進度'
  },
  'Lead handling': {
    'es-ES': 'Gestión de leads', 'fr-FR': 'Gestion des prospects', 'de-DE': 'Lead-Bearbeitung',
    'ja-JP': 'リード対応', 'ko-KR': '리드 처리', 'pt-PT': 'Gestão de leads', 'ar-SA': 'إدارة العملاء المحتملين',
    'ru-RU': 'Обработка лидов', 'it-IT': 'Gestione lead', 'zh-TW': '線索處理'
  },
  'Campaign results': {
    'es-ES': 'Resultados', 'fr-FR': 'Résultats', 'de-DE': 'Ergebnisse',
    'ja-JP': '成果', 'ko-KR': '성과', 'pt-PT': 'Resultados', 'ar-SA': 'النتائج',
    'ru-RU': 'Результаты', 'it-IT': 'Risultati', 'zh-TW': '活動成效'
  },
  'Customer health': {
    'es-ES': 'Salud cliente', 'fr-FR': 'Santé client', 'de-DE': 'Kundengesundheit',
    'ja-JP': '顧客健全性', 'ko-KR': '고객 상태', 'pt-PT': 'Saúde do cliente', 'ar-SA': 'صحة العميل',
    'ru-RU': 'Состояние клиента', 'it-IT': 'Salute cliente', 'zh-TW': '客戶健康度'
  },
  'Project progress': {
    'es-ES': 'Progreso', 'fr-FR': 'Avancement', 'de-DE': 'Projektfortschritt',
    'ja-JP': 'プロジェクト進捗', 'ko-KR': '프로젝트 진행', 'pt-PT': 'Progresso', 'ar-SA': 'تقدم المشروع',
    'ru-RU': 'Ход проекта', 'it-IT': 'Avanzamento', 'zh-TW': '專案進度'
  },
  'Product inventory': {
    'es-ES': 'Inventario', 'fr-FR': 'Stock', 'de-DE': 'Bestand',
    'ja-JP': '在庫', 'ko-KR': '재고', 'pt-PT': 'Inventário', 'ar-SA': 'المخزون',
    'ru-RU': 'Склад', 'it-IT': 'Inventario', 'zh-TW': '產品庫存'
  },
  'Admin settings': {
    'es-ES': 'Administración', 'fr-FR': 'Administration', 'de-DE': 'Administration',
    'ja-JP': '管理設定', 'ko-KR': '관리 설정', 'pt-PT': 'Administração', 'ar-SA': 'الإدارة',
    'ru-RU': 'Администрирование', 'it-IT': 'Amministrazione', 'zh-TW': '管理設定'
  },
  'Analytics': {
    'es-ES': 'Analítica', 'fr-FR': 'Analyse', 'de-DE': 'Analysen',
    'ja-JP': '分析', 'ko-KR': '분석', 'pt-PT': 'Análises', 'ar-SA': 'التحليلات',
    'ru-RU': 'Аналитика', 'it-IT': 'Analisi', 'zh-TW': '分析'
  },
  'Contact details': {
    'es-ES': 'Contacto', 'fr-FR': 'Contact', 'de-DE': 'Kontakt',
    'ja-JP': '連絡先', 'ko-KR': '연락처', 'pt-PT': 'Contacto', 'ar-SA': 'جهة الاتصال',
    'ru-RU': 'Контакты', 'it-IT': 'Contatto', 'zh-TW': '聯絡資料'
  },
  'Help': {
    'es-ES': 'Ayuda', 'fr-FR': 'Aide', 'de-DE': 'Hilfe',
    'ja-JP': 'ヘルプ', 'ko-KR': '도움말', 'pt-PT': 'Ajuda', 'ar-SA': 'مساعدة',
    'ru-RU': 'Справка', 'it-IT': 'Aiuto', 'zh-TW': '說明'
  }
}

function localizeByFallbackTerms(text, locale) {
  const targetLocale = locale === 'zh-CN' || locale === 'zh-TW' ? locale : 'en-US'
  let output = String(text || '').trim()
  if (!output) return ''
  if (/^nav\./.test(output)) output = output.replace(/^nav\./, '')
  output = output
    .replace(/[「」“”]/g, '')
    .replace(/[，、；;]+/g, ', ')
    .replace(/[。]+/g, '. ')
    .replace(/[：:]+/g, ': ')
    .replace(/\s+/g, ' ')
    .trim()
  const entries = Object.entries(englishFallbackTerms).sort((a, b) => b[0].length - a[0].length)
  for (const [zh, en] of entries) {
    output = output.replaceAll(zh, en)
  }
  output = output
    .replace(/[A-Za-z0-9_./:-]+\s*=>\s*/g, '')
    .replace(/[^\x00-\x7F]+/g, ' ')
    .replace(/\s{2,}/g, ' ')
    .trim()
  output = output
    .replace(/\s+,/g, ',')
    .replace(/,\s*,/g, ',')
    .replace(/^,\s*/, '')
    .replace(/,\s*$/, '')
    .replace(/\s+\./g, '.')
    .replace(/\s*\/\s*$/g, '')
    .replace(/\s+:/g, ':')
    .replace(/\s{2,}/g, ' ')
    .trim()
  if (!output || output === text || looksLikeBrokenFallback(output)) {
    const simple = compactFallbackFromText(text, locale)
    return isStrictLocaleValueUsable(simple, locale) ? simple : ''
  }
  return isStrictLocaleValueUsable(output, targetLocale) ? output : ''
}

function looksLikeBrokenFallback(text) {
  const value = String(text || '').trim()
  if (!value) return true
  if (/^\/?$/.test(value)) return true
  if (/,,|,\.|\.\s*,|^,|\/$/.test(value) && value.length < 120) return true
  return /\b[A-Za-z]+Create,|CollaboratePermission|Planned completionTime|Handle,\.|Nombre de \/|Amount,,|Order,,|Contact, Time,|Platform Draft,\.|Customer, Lead, Opportunity, Order,\./.test(value)
}

function withKeySuffix(text, key) {
  const suffix = String(key || '').replace(/^domain\.auto\./, '').slice(0, 4)
  return suffix ? `${text} ${suffix}` : text
}

function humanizeFallbackKey(key, locale) {
  const raw = String(key || '').trim()
  if (!raw) return generic[locale] || strictTranslationMissingByLocale[locale] || ''
  if (/^domain\.auto\.[a-f0-9]+$/i.test(raw)) return ''
  const cleaned = raw
    .replace(/^domain\.auto\.[a-f0-9]+$/i, '')
    .replace(/^(domain|navDesc|nav|menuGroup|app|campaign|im)\./, '')
    .replace(/[._-]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
  if (!cleaned) return compactFallbackFromKey(raw, locale)
  if (locale === 'zh-CN') return cleaned
  if (locale === 'zh-TW') return toTraditionalText(cleaned)
  return cleaned.replace(/\b\w/g, (letter) => letter.toUpperCase())
}

function compactFallbackFromText(text, locale) {
  const normalized = String(text || '').trim()
  if (!normalized) return ''
  if (!hasHanText(normalized)) return normalized
  const compact = normalized
    .replace(/[^\x00-\x7F]+/g, ' ')
    .replace(/\s{2,}/g, ' ')
    .trim()
  if (compact) return compact
  const terms = [
    [/权限|访问|公开|私密|负责人|协作/, 'Access'],
    [/通知|消息|提醒|待办/, 'Notice'],
    [/线索|培育|SDR|MQL|SQL/, 'Lead'],
    [/商机|报价|赢单|输单|成交|交易/, 'Deal'],
    [/订单|回款|发货|发票|合同/, 'Order'],
    [/活动|归因|渠道|UTM|ROI|预算/, 'Campaign'],
    [/客户|公司|联系人/, 'Customer'],
    [/项目|里程碑|交付|验收/, 'Project'],
    [/工单|SLA|报错|截图/, 'Ticket'],
    [/服务/, 'Service'],
    [/产品|库存|SKU/, 'Product'],
    [/账号|用户|角色|部门|岗位|系统|平台|租户/, 'Admin'],
    [/统计|大屏|漏斗|趋势|排行|洞察/, 'Analytics'],
    [/文件|附件|素材|资料|上传/, 'Files'],
    [/说明|建议|帮助|提示/, 'Help']
  ]
  for (const [re, label] of terms) {
    if (re.test(normalized)) return localizeCompactEnglish(label, locale)
  }
  return ''
}

function compactFallbackFromKey(key, locale) {
  return ''
}
