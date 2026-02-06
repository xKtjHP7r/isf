import i18n from '@/core/i18n';

export default i18n([
    [
        '取消',
        '取消',
        'Cancel',
    ],
    [
        '域名：',
        '網域名稱：',
        'Domain Name: ',
    ],
    [
        '定期同步：',
        '定期同步：',
        'Scheduled Sync:',
    ],
    [
        '同步源：',
        '同步源：',
        'Source:',
    ],
    [
        '开启',
        '開啟',
        'Enabled',
    ],
    [
        '关闭',
        '關閉',
        'Disabled',
    ],
    [
        '选择',
        '選取',
        'Select',
    ],
    [
        '同步目标：',
        '同步目標：',
        'Target:',
    ],
    [
        '默认同步到以域控命名的新组织',
        '默認同步到以網域控命名的新組織',
        'Default Sync to the new Domain Org.',
    ],
    [
        '同步周期：',
        '同步週期',
        'Sync Interval:',
    ],
    [
        '分钟',
        '分鐘',
        'min(s)',
    ],
    [
        '小时',
        '小時',
        'hour(s)',
    ],
    [
        '天',
        '天',
        'day(s)',
    ],
    [
        '用户默认状态：',
        '使用者預設狀態',
        'User Status:',
    ],
    [
        '启用',
        '啟用',
        'Enabled',
    ],
    [
        '禁用',
        '停用',
        'Disabled',
    ],
    [
        '同步方式：',
        '同步方式：',
        'Sync Method:',
    ],
    [
        '同步选中的对象及其成员（包括上层的组织结构）',
        '同步選中的物件及其成員（包括上層的組織結構）',
        'Sync the selected objects and the members, including the superior departments.',
    ],
    [
        '同步选中的对象及其成员（不包括上层的组织结构）',
        '同步選中的物件及其成員（不包括上層的組織結構）',
        'Sync the selected objects and the members, excluding the superior departments.',
    ],
    [
        '仅同步用户账号（不包括组织结构）',
        '僅同步使用者（不包括組織結構）',
        'Sync user accounts only, excluding the organization info.',
    ],
    [
        '用户有效期限：',
        '使用者有效期間：',
        'Expires:',
    ],
    [
        '永久有效',
        '永久有效',
        'Never',
    ],
    [
        '一个月',
        '一個月',
        '1 month',
    ],
    [
        '三个月',
        '三個月',
        '3 months',
    ],
    [
        '半年',
        '半年',
        '6 months',
    ],
    [
        '一年',
        '一年',
        '1 year',
    ],
    [
        '两年',
        '兩年',
        '2 years',
    ],
    [
        '三年',
        '三年',
        '3 years',
    ],
    [
        '四年',
        '四年',
        '4 years',
    ],
    [
        '同步关键字设置',
        '同步關鍵字設定',
        'Sync Settings',
    ],
    [
        '保存',
        '儲存',
        'Save',
    ],
    [
        '选择同步目标',
        '選擇同步目標',
        'Select Target Location',
    ],
    [
        '默认以当前域为同步源',
        '預設以當前網域為同步源',
        'Set this domain as sync source by default',
    ],
    [
        '注：修改同步位置后，需将原同步位置的域组织结构从用户组织结构中删除，否则无法同步到新的同步位置。',
        '註：修改同步位置後，需將遠同步位置的域組織結構從使用者組織結構中刪除，否則無法同步到新的同步位置。',
        'Note: You should delete the domain organization in the previous sync place first after you have modified the sync location. Otherwise, the new settings will not be effective.',
    ],
    [
        '已存在相同的域名。',
        '已存在相同的網域名稱。',
        'The same domain name already exists.',
    ],
    [
        '该域不存在。',
        '該網域不存在。',
        'The domain does not exists.',
    ],
    [
        '当前选择的域：',
        '當前選擇的網域：',
        'Current Domain:',
    ],
    [
        '主域',
        '主網域',
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
        '编辑',
        '編輯',
        'Edit',
    ],
    [
        '域名“${name}”不正确。',
        '域名“${name}”不正确。',
        '域名“${name}”不正确。',
    ],
    [
        '请输入1-60的数值',
        '請輸入1-60的整數',
        'Enter an integer from 1 to 60',
    ],
    [
        '请输入1-24的数值',
        '請輸入1-24的整數',
        'Enter an integer from 1 to 24',
    ],
    [
        '请输入正整数',
        '請輸入正整數	',
        'Enter a positive integer',
    ],
    [
        '关闭 域控 “${name}” 定期同步 成功',
        '關閉 網域控制站 “${name}” 定期同步 成功',
        'Disable Scheduled Sync of domain "${name}" successfully',
    ],
    [
        '开启 域控 “${name}” 定期同步 成功',
        '開啟 網域控制站 “${name}” 定期同步 成功',
        'Enable Scheduled Sync of domain "${name}" successfully',
    ],
    [
        '同步目标的部门“${name}”不存在。',
        '同步目標的部門“${name}”不存在。',
        'Dept. "${name}" does not exist in Sync Target.',
    ],
    [
        '域控制器 “${domainName}” 已不存在。',
        '域網控制器 “${domainName}” 已不存在。',
        'Domain Controller "${domainName}" does not exist.',
    ],
    [
        '连接LDAP服务器失败，请检查域控制器地址是否正确，或者域控制器是否开启。',
        '連接LDAP伺服器失敗，請檢查網域控制器位址是否正確，或者網域控制器是否已開啟。',
        'Failed to connect the LDAP server. Please ensure that your controller address is correct and the controller is enabled. ',
    ],
    [
        '设置 域 “${name}” 同步目标为 “${ouPath}”',
        '設定 網域 “${name}” 同步目標為 “${ouPath}”',
        'Set Domain the sync target of "${name}" as "${ouPath}"',
    ],
    [
        '新建用户密级：',
        '新使用者密級：',
        'New User Security Level: ',
    ],
    [
        '同步周期 “${syncInterval}”；新建用户密级 “${csfLevel}”；用户有效期限 “${validPeriod}”；用户默认状态 “${syncStatus}”；同步方式 “${syncMode}”',
        '同步週期 “${syncInterval}”；新增使用者密級 “${csfLevel}”；使用者有效期限 “${validPeriod}”；使用者預設狀態 “${syncStatus}”；同步方式 “${syncMode}”',
        'Sync Interval "${syncInterval}"; Security Level for New Users "${csfLevel}"; Expires "${validPeriod}"; User Status "${syncStatus}"; Sync Method "${syncMode}"',
    ],
    [
        '由 ${oldText} 改为 ${newText}',
        '由 ${oldText} 改為 ${newText}',
        'Changed from ${oldText} to ${newText}',
    ],

])