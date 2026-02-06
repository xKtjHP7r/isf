declare namespace Console {
    namespace SetOrgManager {
        interface Props extends React.Props<void> {
            /**
             * 登录用户id
             */
            userid: string;

            /**
             * 编辑角色
             */
            editRateInfo: Object;

            /**
             * 选择设置的角色
             */
            roleInfo: Object;

            /**
             * 选择配置角色的用户
             */
            userInfo: Object;

            /**
             * 直属部门
             */
            directDeptInfo: Object;

            /**
             * 当前登录用户角色信息
             */
            roles: any;

            /**
             * 确定
             */
            onConfirmSetRoleConfig: () => void;

            /**
             * 取消
             */
            onCancelSetRoleConfig: () => void;
        }

        interface State {
            /**
             * 输入框状态
             */
            validateState: {
                /**
                 * 用户管理最大可分配空间
                 */
                userSpace: number;

                /**
                 * 文档管理最大可分配空间
                 */
                docSpace: number;
            };

            /**
             * 复选框状态
             */
            limitCheckStatus: {
                /**
                 * 用户管理
                 */
                limitUserCheckSatus: boolean;

                /**
                 * 文档管理
                 */
                limitDocCheckSatus: boolean;
            };

            /**
             * 复选框是否禁用
             */
            limitCheckDisable: {
                /**
                 * 用户管理
                 */
                limitUserCheckDisable: boolean;

                /**
                 * 文档管理
                 */
                limitDocCheckDisable: boolean;
            };

            /**
             * 输入框的值
             */
            spaceConfig: {
                /**
                 * 用户管理
                 */
                userSpace: string;

                /**
                 * 文档管理
                 */
                docSpace: string;
            };

            /**
             * 管辖的部门
             */
            selectDeps: Array<any>;

            /**
             * 已选部门的状态
             */
            selectState: boolean;
        }
    }
}