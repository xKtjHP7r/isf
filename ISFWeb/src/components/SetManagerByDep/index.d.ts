declare namespace Console {
    namespace SetManagerByDep {
        interface Props extends React.Props<void> {
            /**
             * 部门Id
             */
            departmentId: string;

            /**
             * 部门名
             */
            departmentName: string;

            /**
             * 用户id
             */
            userid: string;

            /**
             * 设置成功
             */
            onSetSuccess: () => void;

            /**
             * 取消设置
             */
            onCancel: () => void;
        }

        interface State {
            /**
             * 设置组织管理员面板状态
             */
            isConfigManager: boolean;

            /**
             * 选择用户界面
             */
            isAddingManager: boolean;

            /**
             * 组织管理员数据
             */
            managers: ReadonlyArray<any>;

            /**
             * 当前选择的用户
             */
            currentUser: any | null;
            /**
             * 错误
             */
            errorStatus: any;

            /**
             * 正在设置
             */
            isSetting: boolean;

            /**
             * 限额状态多选框是否禁用
             */
            limitCheckDisable: ChexkDisable;
        }

        /**
         * 限额状态多选框是否禁用
         */
        interface ChexkDisable {
            limitUserCheckDisable: boolean;
            limitDocCheckDisable: boolean;
        }
    }
}