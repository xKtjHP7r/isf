import i18n from '@/core/i18n';
export default i18n([
    [
        '新建',
        '新增',
        'Create',
    ],
    [
        '立即同步',
        '立即同步',
        'Sync Now',
    ],
    [
        '编辑',
        '編輯',
        'Edit',
    ],
    [
        '删除',
        '刪除',
        'Delete',
    ],
    [
        '关闭定期同步',
        '關閉定期同步',
        'Disable',
    ],
    [
        '开启定期同步',
        '開啟定期同步',
        'Enable',
    ],
    [
        '域控制器已启用',
        '網域控制站已啟用',
        'Domain Controller is enabled',
    ],
    [
        '禁用域控制器',
        '停用網域控制站',
        'Disable Domain Controller',
    ],
    [
        '当前选择的域：',
        '當前选取的網域：',
        'Current Domain:',
    ],
    [
        '添加域类型：',
        '添加網域型別：',
        'Domain Type:',
    ],
    [
        '主域',
        '主網域',
        'Primary',
    ],
    [
        '域',
        '網域',
        'Domain',
    ],
    [
        '子域',
        '子網域',
        'Subdomain',
    ],
    [
        '信任域',
        '信任網域',
        'Trusted Domain',
    ],
    [
        '定期同步:',
        '定期同步:',
        'Periodic Sync:',
    ],
    [
        '开启',
        '開啟',
        'On',
    ],
    [
        '关闭',
        '關閉',
        'Off',
    ],
    [
        '域名称',
        '網域名稱',
        'Domain Name',
    ],
    [
        '类型',
        '類型',
        'Type',
    ],
    [
        '定期同步',
        '定期同步',
        'Scheduled Sync',
    ],
    [
        '同步周期',
        '同步週期',
        'Sync Interval',
    ],
    [
        '操作',
        '操作',
        'Action',
    ],
    [
        '天',
        '天',
        ' day(s)',
    ],
    [
        '小时',
        '小時',
        ' hour(s)',
    ],
    [
        '分钟',
        '分鐘',
        ' minute(s)',
    ],
    [
        '开启 域控 “${name}” 定期同步 成功',
        '開啟 網域控制站 “${name}” 定期同步 成功',
        'Enable Periodic Sync of domain "${name}" successfully',
    ],
    [
        '关闭 域控 “${name}” 定期同步 成功',
        '關閉 網域控制站 “${name}” 定期同步 成功',
        'Disable Periodic Sync of domain "${name}" successfully',
    ],
    [
        '启用 域控 “${name}” 成功',
        '啟用 網域控制站 “${name}” 成功',
        'Enable domain controller "${name}" successfully',
    ],
    [
        '禁用 域控 “${name}” 成功',
        '停用 網域控制站 “${name}” 成功',
        'Disable domain controller "${name}" successfully',
    ],
    [
        '删除 域控 “${name}” 成功',
        '刪除 網域控制站 “${name}” 成功',
        'Delete domain controller "${name}" successfully',
    ],
    [
        '无法连接文档域',
        '無法連接文件網域',
        'Cannot connect to Document Domain',
    ],
    [
        '此项不允许为空。',
        '必填欄位。',
        'This field is required.',
    ],
    [
        '域名只能包含 英文、数字 及 -. 字符，每一级不能以“-”字符开头或结尾，每一级长度必需 1~63 个字符，且总长不能超过253个字符。',
        '網域名稱只能包含 英文、數字 及 -. 字元， 每一級不能以“-”字元開頭或結尾，每一級長度必須 1~63 個字元，且總長不能超過253個字元。',
        'Domain name can only contain letters, numbers and characters like "-.", and cannot exceed 253 characters. Each segment cannot start, end with "-.=", or exceed 63 characters.',
    ],
    [
        'IP地址输入不合法，请检查您输入的内容是否有误。IPv4地址格式形如 XXX.XXX.XXX.XXX，每段必须是 0~255 之间的整数。 IPv6地址格式形如 XXXX:XXXX:XXXX:XXXX:XXXX:XXXX:XXXX:XXXX，其中每个X都为十六进制数。',
        'IP位址輸入不合法，請檢查您輸入的內容是否有誤。IPv4位址格式形如 XXX.XXX.XXX.XXX，每段必須是 0~255 之間的整數。 IPv6位址格式形如 XXXX:XXXX:XXXX:XXXX:XXXX:XXXX:XXXX:XXXX，其中每個x都為十六進位數。',
        'Input error. Please check your content.The format for IPv4 is XXX.XXX.XXX, in which each segment must be an integer between 0 and 255. The format for IPv6 is XXXX:XXXX:XXXX:XXXX:XXXX:XXXX:XXXX:XXXX, in which X stands for a hexadecimal number.',
    ],
    [
        '端口号必须是1~65535之间的整数。',
        '埠口號必須是1-65535之間的整數。',
        'Port number should be an integer from 1 to 65535.',
    ],
    [
        '禁用域控制器将导致已导入的用户无法验证，您确定要执行此操作吗？',
        '停用網域控制站將導致已匯入的使用者無法驗證，您確定要執行此操作嗎？',
        'If you disable the controller, the imported users will not be verified. Continue?',
    ],
    [
        '是否确定启用“${name}”？',
        '是否確定啟用“${name}”？',
        'Enable "${name}"?',
    ],
    [
        '启用域控制器',
        '啟用域網控制站',
        'Enable Domain Controller',
    ],
    [
        '域控信息将开始同步，请稍后从日志中查询同步结果。',
        '網域控制資訊將開始同步，請稍後從日誌中查詢同步結果。',
        'Sync soon. You can check the sync results in logs later.',
    ],
    [
        '域控制器 “${name}” 已不存在。',
        '域網控制器 “${name}” 已不存在。',
        'Domain Controller "${name}" does not exist.',
    ],
    [
        '删除域控制器将导致已导入用户无法验证，您确定要执行此操作吗？',
        '删除網域控制站將導致已匯入使用者無法驗證，您確定要執行此操作嗎？',
        'If you delete the controller, the imported users will not be verified. Continue?',
    ],
    [
        '当前域控地址与备用域地址相同。',
        '當前網域控制站位址和備用網域位址相同。',
        'This controller address is the same as one of the backup controller.',
    ],
    [
        '当前域控地址与主域不在同一个域内。',
        '當前網域控制站位址和主網域不在同一個網域內。',
        'This controller address is not in the same domain as the primary controller.',
    ],
    [
        '当前域控地址与主域地址相同。',
        '當前網域控制器位址和主網域位址相同。',
        'This controller address is the same as the primary controller.',
    ],
    [
        '当前域控地址已存在。',
        '當前網域控制位址已存在。',
        'This address already exists.',
    ],
    [
        '当前域名已存在。',
        '當前網域名稱已存在。',
        'This domain name already exists.',
    ],
    [
        '域控制器 “${name}” 已不存在。',
        '域網控制器 “${name}” 已不存在。',
        'Domain Controller "${name}" does not exist.',
    ],
    [
        '当前域名与域控地址不匹配。',
        '當前網域名稱與網域位址不匹配。',
        'This domain name does not match with the controller address.',
    ],
])