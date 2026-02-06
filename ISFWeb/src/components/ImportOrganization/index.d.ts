import ImportOrganization from './component.view';

declare namespace Components {
    namespace ImportOrganization {
        interface Props extends React.Props<void> {
            /**
             * 导入的目标部门的id
             */
            departmentId: string;

            /**
             * 导入的操作者
             */
            userid: string;

            /**
             * 导入成功
             */
            onSuccess: () => void;

            /**
             * 导入完成
             */
            onComplete: () => void;

            /**
             * 取消导入
             */
            onCancel: () => void;

        }

        interface State {
            /**
             * 选项配置
             */
            option: Core.ShareMgnt.ncTUsrmImportOption;

            /**
             * 导入方式
             */
            importOption: number;

            /**
             * 选择的数据
             */
            selectedData: any;

            /**
             * 进度
             */
            progress: number;

            /**
             * 错误信息
             */
            failMessage: string;

            /**
             * 错误处理
             */
            errorStatus: number;

            /**
             * 有效期限
             */
            expireTime: number;

            /**
             * 无效的有效期
             */
            invalidExpireTime: boolean;
        }
    }
}