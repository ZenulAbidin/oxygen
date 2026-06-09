import * as React from "react";
import {PageContainer} from "@ant-design/pro-components";
import {Row, Typography, notification, Divider} from "antd";
import {CheckOutlined} from "@ant-design/icons";
import PaymentMethodsSelect from "src/components/payment-methods-select/payment-methods-select";
import WithdrawAddresses from "src/components/withdraw-addresses/withdraw-addresses";
import ApiKeysSection from "src/components/api-keys-section/api-keys-section";
import DevelopersSection from "src/components/developers-section/developers-section";
import PaymentSettingsSection from "src/components/payment-settings-section/payment-settings-section";

const SettingsPage: React.FC = () => {
    const [notificationApi, contextHolder] = notification.useNotification();

    const openNotification = (title: string, description: string) => {
        notificationApi.info({
            message: title,
            description,
            placement: "bottomRight",
            icon: <CheckOutlined style={{color: "#49D1AC"}} />
        });
    };

    return (
        <PageContainer header={{title: ""}}>
            {contextHolder}
            <Row align="middle" justify="space-between">
                <Typography.Title>Settings</Typography.Title>
            </Row>
            <PaymentSettingsSection openPopupFunc={openNotification} />
            <Divider />
            <PaymentMethodsSelect />
            <WithdrawAddresses openPopupFunc={openNotification} />
            <Divider />
            <DevelopersSection openPopupFunc={openNotification} />
            <Divider />
            <ApiKeysSection openPopupFunc={openNotification} />
        </PageContainer>
    );
};

export default SettingsPage;
