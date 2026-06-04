import * as React from "react";
import {Button, Form, InputNumber, Row, Space, Typography} from "antd";
import {SaveOutlined} from "@ant-design/icons";
import useSharedMerchantId from "src/hooks/use-merchant-id";
import useSharedMerchant from "src/hooks/use-merchant";
import merchantProvider from "src/providers/merchant-provider";
import {PaymentSettings} from "src/types";

interface Props {
    openPopupFunc: (title: string, desc: string) => void;
}

const defaultExpirationMinutes = 20;

const PaymentSettingsSection: React.FC<Props> = (props: Props) => {
    const [form] = Form.useForm<PaymentSettings>();
    const {merchantId} = useSharedMerchantId();
    const {merchant, getMerchant} = useSharedMerchant();
    const [isSaving, setIsSaving] = React.useState(false);

    React.useEffect(() => {
        form.setFieldsValue({
            defaultExpirationMinutes:
                merchant?.paymentSettings?.defaultExpirationMinutes ?? defaultExpirationMinutes
        });
    }, [form, merchant?.paymentSettings?.defaultExpirationMinutes]);

    const updatePaymentSettings = async (values: PaymentSettings) => {
        if (!merchantId) {
            return;
        }

        try {
            setIsSaving(true);
            await merchantProvider.updateMerchantPaymentSettings(merchantId, values);
            await getMerchant(merchantId);
            props.openPopupFunc("Payment settings updated", "Default expiration time has been updated");
        } catch (error) {
            console.error("error occurred: ", error);
        } finally {
            setIsSaving(false);
        }
    };

    return (
        <>
            <Row align="middle" justify="space-between">
                <Typography.Title level={3}>Payment settings</Typography.Title>
            </Row>
            <Form<PaymentSettings> form={form} layout="vertical" onFinish={updatePaymentSettings}>
                <Form.Item
                    label="Default invoice expiration"
                    name="defaultExpirationMinutes"
                    rules={[
                        {required: true, message: "Field is required"},
                        {type: "number", min: 1, max: 1440, message: "Use 1 to 1440 minutes"}
                    ]}
                    validateTrigger="onBlur"
                >
                    <InputNumber min={1} max={1440} precision={0} addonAfter="minutes" style={{width: 220}} />
                </Form.Item>
                <Form.Item>
                    <Space>
                        <Button
                            disabled={isSaving || !merchantId}
                            loading={isSaving}
                            type="primary"
                            htmlType="submit"
                            icon={<SaveOutlined />}
                        >
                            Save
                        </Button>
                    </Space>
                </Form.Item>
            </Form>
        </>
    );
};

export default PaymentSettingsSection;
