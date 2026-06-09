import "./withdraw-form.scss";

import * as React from "react";
import {v4 as uuidv4} from "uuid";
import bevis from "src/utils/bevis";
import {
    Form,
    Input,
    Button,
    Space,
    Select,
    Typography,
    Row,
    Col,
    Table,
    FormInstance,
    Statistic,
    InputNumber
} from "antd";
import {ColumnsType} from "antd/es/table";
import {BigFloat, set_precision as setPrecision} from "bigfloat.js";
import AddressCreateForm from "src/components/address-create-form/address-create-form";
import {Withdrawal, MerchantBalance, MerchantAddress, ServiceFee, MerchantAddressParams} from "src/types";
import balancesQueries from "src/queries/balances-queries";
import {useMount} from "react-use";
import SpinWithMask from "src/components/spin-with-mask/spin-with-mask";
import DrawerForm from "src/components/drawer-form/drawer-form";
import {sleep} from "src/utils";
import addressQueries from "src/queries/address-queries";

interface Props {
    balance: MerchantBalance;
    onCancel: () => void;
    onFinish: (values: Withdrawal, form: FormInstance<Withdrawal>) => Promise<void>;
    addresses: MerchantAddress[];
    openPopupFunc: (title: string, description: string) => void;
    isFormSubmitting: boolean;
}

const b = bevis("withdraw-form");

interface WithdrawDetailsItem {
    label: string;
    price: string;
    convertedPrice: string;
    id: string;
}

const columns: ColumnsType<WithdrawDetailsItem> = [
    {
        title: "Label",
        dataIndex: "label",
        key: "label",
        width: "min-content",
        render: (_, record) => <span>{record.label}</span>
    },
    {
        title: "Price",
        dataIndex: "price",
        key: "price",
        render: (_, record) => <span>{record.price}</span>
    },
    {
        title: "Converted price",
        dataIndex: "convertedPrice",
        key: "convertedPrice",
        render: (_, record) => <span>{record.convertedPrice}</span>
    }
];

const WithdrawForm: React.FC<Props> = (props: Props) => {
    const getExchangeRate = balancesQueries.getExchangeRate();
    const getServiceFee = balancesQueries.getServiceFee();
    const createAddress = addressQueries.createAddress();
    const [form] = Form.useForm<Withdrawal>();
    const [amount, changeAmount] = React.useState<string>("");
    const [convertedAmount, setConvertedAmount] = React.useState<string>("");
    const [availableBalance, setAvailableBalance] = React.useState<string>("");
    const [fee, setFee] = React.useState<ServiceFee>();
    const [isFormOpen, changeIsFormOpen] = React.useState<boolean>(false);
    const nullAmount = React.useMemo(() => new BigFloat(0), []);
    const minWithdrawalAmount = React.useMemo(() => fee?.minimumAmount ?? nullAmount.toString(), [
        fee?.minimumAmount,
        nullAmount
    ]);
    const maxWithdrawalAmount = React.useMemo(() => {
        if (props.balance.amount === "empty") {
            return nullAmount.toString();
        }

        return fee?.maximumAmount ?? props.balance.amount;
    }, [fee?.maximumAmount, nullAmount, props.balance.amount]);

    const setWithdrawalAmount = React.useCallback(
        (nextAmount: string) => {
            changeAmount(nextAmount);
            form.setFieldValue("amount", nextAmount);
        },
        [form]
    );

    const balanceId = `${props.balance.blockchainName} ${props.balance.ticker} ${
        props.balance.isTest ? "⚠️ testnet balance" : ""
    }`;

    const loadServiceFee = async () => {
        if (props.balance.id !== "empty") {
            await getServiceFee.mutateAsync(props.balance.id);
        }
    };

    function isNumeric(value: string) {
        return /^-?\d+$/.test(value);
    }

    function displayAmount(amount: string) {
        return props.balance.isTest ? "0" : amount;
    }

    const checkCorrectAmount = () => {
        if (!amount || amount[0] === "-") {
            return false;
        }

        const splitted = amount.split(".");

        if (splitted.length === 1) {
            if (!isNumeric(splitted[0])) {
                return false;
            }

            return true;
        }

        if (!isNumeric(splitted[0]) || !isNumeric(splitted[1]) || splitted[1] === "") {
            return false;
        }

        return true;
    };

    const loadConvertedAmount = async (amount: string) => {
        if (props.balance.id === "empty" || !checkCorrectAmount()) {
            return;
        }

        setConvertedAmount("");

        const response = await getExchangeRate.mutateAsync({from: props.balance.ticker, amount, to: "USD"});
        setConvertedAmount(response.convertedAmount);
    };

    const loadAvailableBalance = async () => {
        if (props.balance.id === "empty") {
            return;
        }

        setAvailableBalance("");

        const response = await getExchangeRate.mutateAsync({
            from: props.balance.ticker,
            amount: props.balance.amount,
            to: "USD"
        });
        setAvailableBalance(displayAmount(response.convertedAmount));
    };

    const normalizeAmount = () => {
        const convertedAmount = new BigFloat(0).add(amount);
        const maxAmount = new BigFloat(maxWithdrawalAmount);
        let res = amount;

        if (convertedAmount.lessThan(nullAmount)) {
            setWithdrawalAmount(nullAmount.toString());
            res = nullAmount.toString();
        } else if (convertedAmount.greaterThan(maxAmount)) {
            setWithdrawalAmount(maxAmount.toString());
            res = maxAmount.toString();
        }

        return res;
    };

    const applyMaxAmount = () => {
        setWithdrawalAmount(maxWithdrawalAmount);
    };

    const isAmountWithinAllowedRange = () => {
        if (!checkCorrectAmount()) {
            return false;
        }

        const selectedAmount = new BigFloat(0).add(amount);

        return (
            selectedAmount.greaterThan(nullAmount) &&
            !selectedAmount.lessThan(new BigFloat(minWithdrawalAmount)) &&
            !selectedAmount.greaterThan(new BigFloat(maxWithdrawalAmount))
        );
    };

    useMount(async () => {
        await loadServiceFee();
        await loadAvailableBalance();
    });

    React.useEffect(() => {
        if (props.balance.id === "empty") {
            setConvertedAmount("");
            setWithdrawalAmount("");
        } else {
            loadServiceFee();
            loadAvailableBalance();

            form.setFieldValue("balanceId", balanceId);
        }
    }, [props.balance]);

    React.useEffect(() => {
        if (checkCorrectAmount()) {
            const normilizedValue = normalizeAmount();
            loadConvertedAmount(normilizedValue);
        } else {
            setConvertedAmount("");
        }
    }, [amount]);

    React.useEffect(() => {
        if (!getServiceFee.isLoading) {
            setFee(getServiceFee.data);
        }
    }, [getServiceFee.isLoading]);

    const renderTotalSum = () => {
        if (props.balance.amount === "empty" || !fee) {
            return "loading...";
        }

        const total = new BigFloat(0).add(amount).add(fee.currencyFee);
        const splitedTotal = total.toString().split(".");

        const amountSplitByDot = amount.split(".");
        const feeSplitByDot = fee.currencyFee.split(".");
        const amountDigitsAfterDot = amountSplitByDot.length > 1 ? amountSplitByDot[1].length : 0;
        const feeDigitsAfterDot = feeSplitByDot.length > 1 ? feeSplitByDot[1].length : 0;
        const digitsAfterDot = Math.max(amountDigitsAfterDot, feeDigitsAfterDot);

        if (digitsAfterDot === 0) {
            return splitedTotal[0];
        }

        setPrecision(-digitsAfterDot);
        return total.toString();
    };

    const onSubmit = async (values: Withdrawal) => {
        if (props.balance.amount === "empty" || !fee || !isAmountWithinAllowedRange()) {
            return;
        }

        values.balanceId = props.balance.id;

        await props.onFinish(values, form);
    };

    const renderTotalConvertedAmount = () => {
        if (!convertedAmount || !fee) {
            return;
        }

        if (fee.isTest) {
            return "0";
        }

        const total = new BigFloat(0).add(convertedAmount).add(fee.usdFee);

        return total.toString();
    };

    const filterAddresses = () => {
        return props.addresses.filter((item) => item.blockchain === props.balance.blockchain);
    };

    const uploadCreatedAddress = async (value: MerchantAddressParams, form: FormInstance<MerchantAddressParams>) => {
        try {
            await createAddress.mutateAsync(value);
            changeIsFormOpen(false);
            props.openPopupFunc("Address has created", `You have created new address ${value.name}`);

            await sleep(1000);
            form.resetFields();
        } catch (error) {
            console.error("error occurred: ", error);
        }
    };

    const getPaymentsDetails = () => {
        if (!checkCorrectAmount() || !fee) {
            return [
                {
                    label: "You will receive",
                    price: "-",
                    convertedPrice: "-",
                    id: "0"
                },
                {
                    label: "Withdrawal fee",
                    price: "-",
                    convertedPrice: "-",
                    id: "1"
                },
                {
                    label: "Total",
                    price: "-",
                    convertedPrice: "-",
                    id: "2"
                }
            ] as WithdrawDetailsItem[];
        }

        return [
            {
                label: "You will receive",
                price: `${amount} ${props.balance.ticker}`,
                convertedPrice: `$${displayAmount(convertedAmount)}`,
                id: "0"
            },
            {
                label: "Withdrawal fee",
                price: `${fee.currencyFee} ${fee.currency}`,
                convertedPrice: `$${fee.usdFee}`,
                id: "1"
            },
            {
                label: "Total",
                price: `${renderTotalSum()} ${props.balance.ticker}`,
                convertedPrice: `$${renderTotalConvertedAmount()}`,
                id: "2"
            }
        ] as WithdrawDetailsItem[];
    };

    const isLoading =
        getServiceFee.isLoading || !availableBalance || Boolean(checkCorrectAmount() && (!fee || !convertedAmount));

    const filteredAddresses = filterAddresses();

    return (
        <>
            <Form<Withdrawal> form={form} initialValues={{id: uuidv4()}} onFinish={onSubmit} layout="vertical">
                <SpinWithMask isLoading={isLoading} />

                {!getServiceFee.isLoading && Boolean(filteredAddresses.length) && availableBalance && (
                    <>
                        <Form.Item label="Selected balance" name="balanceId" initialValue={balanceId}>
                            <Input className={b("input-disabled")} disabled />
                        </Form.Item>
                        <Row gutter={[16, 16]} className={b("balance-info")}>
                            <Col>
                                <Statistic
                                    title={`Available balance in ${props.balance.ticker}`}
                                    value={`${props.balance.amount} ${props.balance.ticker}`}
                                />
                            </Col>
                            <Col>
                                <Statistic title="Available balance in USD" value={`$${availableBalance}`} />
                            </Col>
                        </Row>
                        <Form.Item
                            label="Address"
                            name="addressId"
                            required
                            rules={[{required: true, message: "Field is required"}, {validateTrigger: ""}]}
                        >
                            <Select
                                className={b("address-select")}
                                options={filteredAddresses.map((address) => ({
                                    value: address.id,
                                    label: `${address.name} (${address.address})`
                                }))}
                            />
                        </Form.Item>
                        <Form.Item required label="Amount" name="amount" className={b("currency-wrap")}>
                            <Space.Compact>
                                <Select
                                    defaultValue={props.balance.ticker}
                                    options={[{value: props.balance.ticker, label: props.balance.ticker}]}
                                    className={b("currency-selection")}
                                    disabled
                                    showArrow={false}
                                />
                                <InputNumber
                                    stringMode
                                    value={amount === "" ? undefined : amount}
                                    onChange={(value) => setWithdrawalAmount(value == null ? "" : String(value))}
                                    min={minWithdrawalAmount}
                                    max={maxWithdrawalAmount}
                                    className={b("currency-input")}
                                />
                                <Button
                                    type="default"
                                    onClick={applyMaxAmount}
                                    disabled={!fee || !new BigFloat(maxWithdrawalAmount).greaterThan(nullAmount)}
                                    className={b("currency-max-button")}
                                >
                                    Max
                                </Button>
                            </Space.Compact>
                        </Form.Item>
                        {fee && (
                            <Typography.Text type="secondary" className={b("note-text")}>
                                {`Minimum withdrawal amount for ${props.balance.ticker}: `}
                                <Typography.Text strong>
                                    {`${fee.minimumAmount} ${fee.currency}`}
                                </Typography.Text>
                                {". "}
                                {"Maximum currently spendable: "}
                                <Typography.Text strong>
                                    {`${fee.maximumAmount} ${fee.currency}`}
                                </Typography.Text>
                            </Typography.Text>
                        )}
                        <Table
                            columns={columns}
                            dataSource={getPaymentsDetails()}
                            rowKey={(item) => item.id}
                            pagination={false}
                            loading={isLoading}
                            showHeader={false}
                            size="middle"
                            className={b("table")}
                        />

                        <Space>
                            <Button type="primary" htmlType="submit" disabled={!fee || !isAmountWithinAllowedRange()}>
                                Create Withdrawal
                            </Button>
                        </Space>
                    </>
                )}

                {!filteredAddresses.length && props.balance.id !== "empty" && (
                    <>
                        <Row>
                            <Typography.Text>
                                To create a withdrawal, first create an address in settings section or you can do it
                                here
                            </Typography.Text>
                        </Row>
                        <br />
                        <Button
                            disabled={props.isFormSubmitting}
                            loading={props.isFormSubmitting}
                            type="primary"
                            onClick={() => changeIsFormOpen(true)}
                        >
                            Create an address
                        </Button>
                    </>
                )}
            </Form>
            <DrawerForm
                title="Create an address"
                isFormOpen={isFormOpen}
                changeIsFormOpen={changeIsFormOpen}
                formBody={
                    <AddressCreateForm
                        isFormSubmitting={props.isFormSubmitting}
                        onCancel={() => {
                            changeIsFormOpen(false);
                        }}
                        onFinish={uploadCreatedAddress}
                    />
                }
            />
        </>
    );
};

export default WithdrawForm;
