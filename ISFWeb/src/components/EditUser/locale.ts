import i18n from '@/core/i18n';

export default i18n([
    [
        '编辑用户',
        '編輯用戶',
        'Edit User',
    ],
    [
        '用户名：',
        '用戶名稱：',
        'Username:',
    ],
    [
        '显示名：',
        '顯示名：',
        'Display Name:',
    ],
    [
        '用户编码：',
        '用戶編碼：',
        'User Code:',
    ],
    [
        '用户唯一标识，全局唯一，如：工号。',
        '用戶唯一標識，全域唯一，如：工號。',
        'Unique User Identifier (Globally Unique, e.g., Employee ID)',
    ],
    [
        '直属上级：',
        '直屬上級：',
        'Direct Supervisor:',
    ],
    [
        '选择',
        '選擇',
        'Select',
    ],
    [
        '岗位：',
        '崗位：',
        'Position:',
    ],
    [
        '选择用户',
        '選擇用戶',
        'Select User',
    ],
    [
        '备注：',
        '備註：',
        'Note:',
    ],
    [
        '直属部门：',
        '直屬部門：',
        'Department:',
    ],
    [
        '认证类型：',
        '驗證類型：',
        'Auth Type:',
    ],
    [
        '邮箱地址：',
        '電子郵件：',
        'Email:',
    ],
    [
        '手机号：',
        '手機號：',
        'Mobile:',
    ],
    [
        '身份证号：',
        '身份證號：',
        'ID Number:',
    ],
    [
        '用户密级：',
        '用戶密級：',
        'Secret Level:',
    ],
    [
        '用户密级2：',
        '用戶密級2：',
        'Secret Level 2:',
    ],
    [
        '非密',
        '非密',
        'Unclassified',
    ],
    [
        '内部',
        '内部',
        'Internal',
    ],
    [
        '秘密',
        '秘密',
        'Secret',
    ],
    [
        '有效期限：',
        '有效期間：',
        'Expiration:',
    ],
    [
        '当前已占用',
        '當前已佔用',
        'Used:',
    ],
    [
        '本地用户',
        '本機用戶',
        'Local User',
    ],
    [
        '域用户',
        '域用戶',
        'Domain User',
    ],
    [
        '外部用户',
        '外部用戶',
        'External User',
    ],
    [
        '确定',
        '確定',
        'OK',
    ],
    [
        '取消',
        '取消',
        'Cancel',
    ],
    [
        '此项不能为空。',
        '必填欄位。',
        'This field is required.',
    ],
    [
        '邮箱地址只能包含 英文、数字 及 @-_. 字符，格式形如 XXX@XXX.XXX，长度范围 5~100 个字符。',
        '電郵位置只能包含英文，數字及@-_.字元，格式形如XXX@XXX.XXX，長度範圍5~100個字元。',
        'Email address can only contain letters; numbers and @-_. Its format should be as XXX@XXX.XXX; and its length should be between 5~100 characters.',
    ],
    [
        '手机号只能包含 数字，长度范围 1~20 个字符，请重新输入。',
        '手機號只能包含 數字、長度範圍1~20個字元，請重新輸入。',
        'The mobilephone number can only contain numbers; and the length should be between 1~20 characters, please re-enter.',
    ],

    [
        '永久有效',
        '永久有效',
        'Never Expires',
    ],
    [
        '编辑用户“${displayName}(${loginName})”成功',
        '編輯用戶“${displayName}(${loginName})”成功',
        'Edit User “${displayName}(${loginName})” Successfully',
    ],
    [
        '用户名 “${loginName}”；显示名 “${display}”；用户编码 “${code}”；直属上级 “${managerDisplayName}”；岗位 “${position}”；备注 “${remark}”；认证类型 “${userType}”；邮箱地址 “${email}”；手机号 “${telNum}”；用户密级 “${csfLevel}”；有效期限 “${expireTime}”；身份证号 “${idcardNumber}”；',
        '用戶名 “${loginName}”；顯示名 “${display}”；用户编码 “${code}”；直屬上級 “${managerDisplayName}”；崗位 “${position}”；備註 “${remark}”；驗證類型 “${userType}”；電子郵件 “${email}”；手機號 “${telNum}”；用戶密級 “${csfLevel}”； 有效期間 “${expireTime}”；身份證號 “${idcardNumber}”；',
        'User Name "${loginName}"; Display Name "${display}"; User Code "${code}"; Direct Supervisor "${managerDisplayName}"; Position "${position}"; Note "${remark}"; Auth Type: "${userType}"; Email "${email}";Phone "${telNum}"; Secret Level “${csfLevel}”； Expiration “${expireTime}”；ID Number “${idcardNumber}”；',
    ],
    [
        '用户名 “${loginName}”；显示名 “${display}”；用户编码 “${code}”；直属上级 “${managerDisplayName}”；岗位 “${position}”；备注 “${remark}”；认证类型 “${userType}”；邮箱地址 “${email}”；手机号 “${telNum}”；用户密级 “${csfLevel}”；用户密级2 “${csfLevel2}”；有效期限 “${expireTime}”；身份证号 “${idcardNumber}”；',
        '用戶名 “${loginName}”；顯示名 “${display}”；用户编码 “${code}”；直屬上級 “${managerDisplayName}”；崗位 “${position}”；備註 “${remark}”；驗證類型 “${userType}”；電子郵件 “${email}”；手機號 “${telNum}”；用戶密級 “${csfLevel}”； 用戶密級2 “${csfLevel2}”；有效期間 “${expireTime}”；身份證號 “${idcardNumber}”；',
        'User Name "${loginName}"; Display Name "${display}"; User Code "${code}"; Direct Supervisor "${managerDisplayName}"; Position "${position}"; Note "${remark}"; Auth Type: "${userType}"; Email "${email}";Phone "${telNum}"; Secret Level “${csfLevel}”；Secret level 2“${csfLevel2}”； Expiration “${expireTime}”； ID Number “${idcardNumber}”；',
    ],
    [
        '编辑失败，直属部门“${dep}”不存在，请重新选择。',
        '編輯失敗，直屬部門“${dep}”已不存在，請重新選擇。',
        'Edit Failure, Department "${dep}" does not exist. Please reselect.',
    ],
    [
        '该显示名已被占用。',
        '該顯示名已被用戶佔用。',
        'The Display Name already exists.',
    ],
    [
        '该日期已过期，请重新选择。',
        '該日期已過期，請重新選擇。',
        'The date has expired, please reselect.',
    ],
    [
        '该邮箱已被占用。',
        '該電子郵件已被使用。',
        'This Email already exists.',

    ],
    [
        '请输入正确的手机号。',
        '請輸入正確的手機號。',
        'Please enter a valid phone number.',
    ],
    [
        '该手机号已被占用。',
        '該手機號已被佔用。',
        'The phone number already exists.',
    ],
    [
        '请输入正确的身份证号。',
        '請輸入正確的身份證號。',
        'Please enter a valid ID number.',
    ],
    [
        '该身份证号已被占用。',
        '該身份證號已被佔用。',
        'The ID number has been register.',
    ],
    [
        '（离线）',
        '（離線）',
        '(Offline)',
    ],
    [
        '已移除',
        '已移除',
        'Removed',
    ],
    [
        'GB',
        'GB',
        'GB',
    ],
    [
        '当前已使用',
        '當前已使用',
        'Currently used',
    ],
    [
        '存储位置：',
        '儲存位置：',
        'Location: ',
    ],
    [
        '您无法编辑自身账号。',
        '您無法編輯自身帳戶。',
        'You cannot edit your own account.',
    ],
    [
        '由 ${oldText} 改为 ${newText}',
        '由 ${oldText} 改为 ${newText}',
        'Changed from ${oldText} to ${newText}',
    ],
    [
        '由 ${oldText} 改为 ${newText}',
        '由 ${oldText} 改为 ${newText}',
        'Changed from ${oldText} to ${newText}',
    ],
])