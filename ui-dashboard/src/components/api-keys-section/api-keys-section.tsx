import * as React from "react";
import {Button, Space, Row, Typography, Dropdown, Table, Result, FormInstance} from "antd";
import {DeleteOutlined, KeyOutlined, MoreOutlined, PlusOutlined} from "@ant-design/icons";
import {ColumnsType} from "antd/es/table";
import {MerchantToken, WebhookSettings} from "src/types";
import useSharedMerchantId from "src/hooks/use-merchant-id";
import useSharedMerchant from "src/hooks/use-merchant";
import tokenQueries from "src/queries/token-quries";
import DrawerForm from "src/components/drawer-form/drawer-form";
import TokenCreateForm, {TokenCreateFormFields} from "src/components/token-create-form/token-create-form";
import merchantProvider from "src/providers/merchant-provider";
import WebhookSettingsForm from "src/components/webhook-settings-form/webhook-settings-form";
import {sleep} from "src/utils";
import TimeLabel from "src/components/time-label/time-label";
import PasswordLabel from "src/components/password-label/password-label";
import createConfirmPopup from "src/utils/create-confirm-popup";

interface Props {
    openPopupFunc: (title: string, desc: string) => void;
}

const ApiKeysSection: React.FC<Props> = (props: Props) => {
    const listTokens = tokenQueries.listTokens();
    const createToken = tokenQueries.createToken();
    const deleteToken = tokenQueries.deleteToken();
    const {merchantId} = useSharedMerchantId();
    const {merchant, getMerchant} = useSharedMerchant();
    const [tokens, setTokens] = React.useState<MerchantToken[]>();
    const [isWebhookSettingsFormOpen, setIsWebhookSettingsFormOpen] = React.useState<boolean>(false);
    const [isCreatingTokenFormOpen, setIsCreatingTokenFormOpen] = React.useState<boolean>(false);
    const [isFormSubmitting, setIsFormSubmitting] = React.useState<boolean>(false);

    const isLoading = listTokens.isLoading || listTokens.isFetching || deleteToken.isLoading || createToken.isLoading;

    const deleteSelectedToken = async (token: MerchantToken) => {
        try {
            await deleteToken.mutateAsync(token.id);
            props.openPopupFunc("Token deleted", `Deleted API token ${token.name}`);
        } catch (error) {
            console.error("error occurred: ", error);
        }
    };

    const columns: ColumnsType<MerchantToken> = [
        {
            title: "Created at",
            dataIndex: "createdAt",
            key: "createdAt",
            width: "15%",
            render: (_, record) => (
                <Space>
                    <TimeLabel time={record.createdAt} />
                </Space>
            )
        },
        {
            title: "Name",
            dataIndex: "tokenName",
            key: "tokenName",
            width: "20%",
            render: (_, record) => <span style={{whiteSpace: "nowrap"}}>{record.name}</span>
        },
        {
            title: "Token",
            dataIndex: "token",
            key: "token",
            render: (_, record) => (
                <Row align="middle" justify="space-between">
                    <PasswordLabel text={record.token} popupFunc={props.openPopupFunc} />
                    <Dropdown
                        menu={{
                            items: [
                                {
                                    label: (
                                        <Space
                                            onClick={() =>
                                                createConfirmPopup(
                                                    "Delete the token",
                                                    <span>Are you sure to delete this token?</span>,
                                                    () => deleteSelectedToken(record)
                                                )
                                            }
                                        >
                                            <span>Delete</span>
                                            <Button type="text" danger icon={<DeleteOutlined />} />
                                        </Space>
                                    ),
                                    key: "0"
                                }
                            ]
                        }}
                        trigger={["click"]}
                    >
                        <Button type="text" icon={<MoreOutlined style={{fontSize: "150%"}} />} />
                    </Dropdown>
                </Row>
            )
        }
    ];

    React.useEffect(() => {
        setTokens(listTokens.data || []);
    }, [listTokens.data]);

    React.useEffect(() => {
        if (!merchantId) {
            setTokens([]);
            return;
        }

        listTokens.refetch();
        getMerchant(merchantId);
    }, [merchantId]);

    const uploadCreatedToken = async (tokenName: string, form: FormInstance<TokenCreateFormFields>) => {
        try {
            setIsFormSubmitting(true);
            await createToken.mutateAsync(tokenName);
            setIsCreatingTokenFormOpen(false);
            props.openPopupFunc("Token created", `Created API token ${tokenName}`);

            await sleep(1000);
            form.resetFields();
        } catch (error) {
            console.error("error occurred: ", error);
        } finally {
            setIsFormSubmitting(false);
        }
    };

    const updateWebhookSettings = async (settings: WebhookSettings, form: FormInstance<WebhookSettings>) => {
        try {
            setIsFormSubmitting(true);
            await merchantProvider.updateMerchantWebhookSettings(merchantId!, settings);
            setIsWebhookSettingsFormOpen(false);
            props.openPopupFunc("Webhook settings updated", "Updated webhook URL and signing secret");

            await sleep(1000);
            await getMerchant(merchantId!);
            form.resetFields();
        } catch (error) {
            console.error("error occurred: ", error);
        } finally {
            setIsFormSubmitting(false);
        }
    };

    return (
        <>
            <Row align="middle" justify="space-between">
                <div>
                    <Typography.Title level={3}>API tokens</Typography.Title>
                    <Typography.Text type="secondary">
                        Merchant API credentials for external applications.
                    </Typography.Text>
                </div>
                <Space>
                    <Button onClick={() => setIsWebhookSettingsFormOpen(true)} style={{marginTop: 20}}>
                        Manage webhook settings
                    </Button>
                    <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => setIsCreatingTokenFormOpen(true)}
                        style={{marginTop: 20}}
                    >
                        Create a token
                    </Button>
                </Space>
            </Row>
            <Table
                columns={columns}
                dataSource={tokens}
                rowKey={(record) => record.id}
                loading={isLoading}
                pagination={false}
                size="middle"
                locale={{
                    emptyText: (
                        <Result
                            icon={<KeyOutlined />}
                            title="No API tokens"
                            subTitle="Create a token to connect an external application."
                        ></Result>
                    )
                }}
            />
            <DrawerForm
                title="Create a token"
                isFormOpen={isCreatingTokenFormOpen}
                changeIsFormOpen={setIsCreatingTokenFormOpen}
                formBody={
                    <TokenCreateForm
                        onCancel={setIsCreatingTokenFormOpen}
                        onFinish={uploadCreatedToken}
                        isFormSubmitting={isFormSubmitting}
                    />
                }
            />
            <DrawerForm
                title="Webhook settings"
                isFormOpen={isWebhookSettingsFormOpen}
                changeIsFormOpen={setIsWebhookSettingsFormOpen}
                formBody={
                    <WebhookSettingsForm
                        onCancel={setIsWebhookSettingsFormOpen}
                        onFinish={updateWebhookSettings}
                        webhookSettings={merchant?.webhookSettings}
                        isFormSubmitting={isFormSubmitting}
                    />
                }
            />
        </>
    );
};

export default ApiKeysSection;
