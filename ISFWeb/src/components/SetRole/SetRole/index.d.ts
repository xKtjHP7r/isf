declare namespace Console {
    namespace SetRoleComponent {
        interface Props extends React.Props<void> {
            /**
             * 选择的用户
             */
            users: Array<any>;

            /**
             * 选择的部门
             */
            dep: Object;

            /**
             * 登录用户的id
             */
            userid: string;

            /**
             * 结束操作
             */
            onComplete: () => void;
        }

        interface State {
            /**
             * 输入框的值
             */
            value: string;

            /**
             * 搜索结果
             */
            results: Array<any>;

            /**
             * 选择的用户
             */
            userInfo: Object;

            /**
             * 用户已拥有的角色
             */
            ownRole: Array<any>;

            /**
             * 登录用户所有可操作的角色
             */
            allSelectableRoles: Array<any>;

            /**
             * 点选给用户配置的角色
             */
            selectRoleInfo: Object;

            /**
             * 是否显示配置角色的设置界面
             */
            showRoleEditDialog: boolean;

            /**
             * 所有可配置的角色，用于实现ASE产品用户已经拥有的共享审核员和定密审核员可编辑和删除，
             * 从老版本升级到ASE，用户已经拥有的共享审核员和定密审核员不屏蔽
             */
            allRoles: Array<any>;

            /**
             * 当前登录用户角色信息
             */
            roles: ReadonlyArray<any>;
        }
    }
}