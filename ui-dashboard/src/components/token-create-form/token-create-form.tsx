import * as React from "react";
import {Form, Input, Button, Space, FormInstance} from "antd";
import {sleep} from "src/utils";

export interface TokenCreateFormFields {
    name: string;
}

interface Props {
    onCancel: (value: boolean) => void;
    onFinish: (values: string, form: FormInstance<TokenCreateFormFields>) => Promise<void>;
    isFormSubmitting: boolean;
}

const TokenCreateForm: React.FC<Props> = (props: Props) => {
    const [form] = Form.useForm<TokenCreateFormFields>();

    const onSubmit = async (values: TokenCreateFormFields) => {
        await props.onFinish(values.name, form);
    };

    const onCancel = async () => {
        props.onCancel(false);

        await sleep(1000);
        form.resetFields();
    };

    return (
        <Form<TokenCreateFormFields> form={form} onFinish={onSubmit} layout="vertical">
            <div>
                <Form.Item
                    label="Token name"
                    name="name"
                    rules={[{required: true, message: "Field is required"}]}
                    validateTrigger="onBlur"
                >
                    <Input style={{width: 300}} placeholder="Production integration" />
                </Form.Item>
            </div>
            <Space>
                <Button
                    disabled={props.isFormSubmitting}
                    loading={props.isFormSubmitting}
                    type="primary"
                    htmlType="submit"
                >
                    Save
                </Button>
                <Button danger onClick={onCancel}>
                    Cancel
                </Button>
            </Space>
        </Form>
    );
};

export default TokenCreateForm;
