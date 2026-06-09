import * as React from "react";
import {v4 as uuidv4} from "uuid";
import {Form, Input, Button, Space, Select, InputNumber, FormInstance, Radio} from "antd";
import {PaymentLinkParams, CURRENCY, PaymentLinkAction, PaymentLinkType} from "src/types";
import {sleep} from "src/utils";
import LinkInput from "src/components/link-input/link-input";

interface Props {
    onCancel: () => void;
    onFinish: (values: PaymentLinkParams, form: FormInstance<PaymentLinkParams>) => Promise<void>;
    isFormSubmitting: boolean;
}

const minPrice = 0.01;
const maxPrice = 10 ** 7;
const linkPrefix = "https://";

const PaymentLinkForm: React.FC<Props> = (props: Props) => {
    const [form] = Form.useForm<PaymentLinkParams>();
    const [linkAction, changeLinkAction] = React.useState<PaymentLinkAction>("showMessage");
    const [linkType, changeLinkType] = React.useState<PaymentLinkType>("payment");

    const onSubmit = async (values: PaymentLinkParams) => {
        if (linkAction === "redirect") {
            values.redirectUrl = linkPrefix + values.redirectUrl;
            values.successMessage = undefined;
        } else {
            values.redirectUrl = undefined;
        }

        values.successAction = linkAction;
        values.type = linkType;
        if (linkType === "donation") {
            values.price = undefined;
        }

        await props.onFinish(values, form);
    };

    const onCancel = async () => {
        props.onCancel();

        await sleep(1000);
        form.resetFields();
    };

    return (
        <Form<PaymentLinkParams>
            form={form}
            initialValues={{id: uuidv4(), type: "payment", currency: "USD"}}
            onFinish={onSubmit}
            layout="vertical"
        >
            <Form.Item required rules={[{required: true, message: "Field is required"}]} label="Name" name="name">
                <Input placeholder="My new link" />
            </Form.Item>
            <Form.Item label="Link type" name="type">
                <Radio.Group
                    value={linkType}
                    optionType="button"
                    buttonStyle="solid"
                    onChange={(event) => {
                        const nextType = event.target.value as PaymentLinkType;
                        changeLinkType(nextType);
                        if (nextType === "donation") {
                            form.setFieldValue("price", undefined);
                        }
                    }}
                >
                    <Radio.Button value="payment">Fixed amount</Radio.Button>
                    <Radio.Button value="donation">Donation</Radio.Button>
                </Radio.Group>
            </Form.Item>
            {linkType === "payment" ? (
                <Space>
                    <Form.Item
                        label="Price"
                        name="price"
                        required
                        rules={[
                            {required: true, message: "Field is required"},
                            {
                                type: "number",
                                message: "Incorrect number value"
                            }
                        ]}
                        validateFirst
                        validateTrigger={["onChange", "onBlur"]}
                    >
                        <InputNumber style={{width: "100%"}} precision={2} min={minPrice} max={maxPrice} />
                    </Form.Item>
                    <Form.Item name="currency" style={{width: 80, marginTop: "30px"}} initialValue="USD">
                        <Select
                            style={{width: 80}}
                            options={CURRENCY.map((currency) => ({value: currency, label: currency}))}
                        />
                    </Form.Item>
                </Space>
            ) : (
                <Form.Item label="Currency" name="currency" initialValue="USD" style={{width: 120}}>
                    <Select style={{width: 120}} options={CURRENCY.map((currency) => ({value: currency, label: currency}))} />
                </Form.Item>
            )}
            <Form.Item label="Description" name="description" style={{width: 300}}>
                <Input.TextArea placeholder="Your description" rows={2} />
            </Form.Item>
            <Form.Item label="Choose action after payment" style={{width: "250px"}}>
                <Select
                    defaultValue={"showMessage"}
                    options={[
                        {
                            value: "showMessage",
                            label: "Show message"
                        },
                        {
                            value: "redirect",
                            label: "Redirect customer"
                        }
                    ]}
                    onChange={(value: PaymentLinkAction) => changeLinkAction(value)}
                />
            </Form.Item>

            {linkAction === "redirect" ? (
                <LinkInput
                    placeholder="your-store.com/success"
                    label="Customer redirect URL"
                    name="redirectUrl"
                    required
                />
            ) : (
                <Form.Item
                    required
                    rules={[{required: true, message: "Field is required"}]}
                    label="Success message"
                    name="successMessage"
                    style={{width: 300}}
                >
                    <Input.TextArea placeholder="Your description" rows={2} />
                </Form.Item>
            )}

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

export default PaymentLinkForm;
