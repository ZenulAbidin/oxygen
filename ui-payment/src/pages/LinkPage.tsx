import * as React from "react";
import {useNavigate, useLocation} from "react-router-dom";
import Icon from "src/components/Icon";
import LoadingTextIcon from "src/components/LoadingTextIcon";
import {usePaymentLink} from "src/hooks/linkContext";
import {usePayment} from "src/hooks/paymentContext";
import paymentProvider from "src/providers/paymentProvider";
import renderCurrency from "src/utils/renderCurrency";

const LinkPage: React.FC = () => {
    const navigate = useNavigate();
    const location = useLocation();
    const {paymentLink} = usePaymentLink();
    const {setPayment} = usePayment();
    const id = React.useRef(location.pathname.match(/\/([^/]+)$/)?.[1]);
    const [donationAmount, setDonationAmount] = React.useState("");
    const [donationAmountError, setDonationAmountError] = React.useState("");

    const createLink = async () => {
        if (!paymentLink || !id.current) {
            return;
        }

        const isDonation = paymentLink.type === "donation";
        let paymentParams: {price: number} | undefined;

        if (isDonation) {
            const amount = Number(donationAmount);
            if (!donationAmount || !Number.isFinite(amount) || amount <= 0) {
                setDonationAmountError("Enter a donation amount");
                return;
            }

            paymentParams = {price: amount};
        }

        try {
            const paymentId = await paymentProvider.createPaymentFromLink(id.current, paymentParams);
            navigate(`/pay/${paymentId}`);
            const payment = await paymentProvider.getPayment(paymentId);
            setPayment(payment);
        } catch {
            navigate("/not-found");
        }
    };

    const isDonation = paymentLink?.type === "donation";

    return (
        <>
            {paymentLink ? (
                <div className="relative">
                    <div className="mx-auto h-16 w-16 flex items-center justify-center mb-2.5 sm:mb-2">
                        <div className="shrink-0">
                            <Icon name="store" className="h-16 w-16" />
                        </div>
                    </div>
                    <span
                        className={`block mx-auto text-2xl font-medium text-center ${
                            paymentLink?.description ? "mb-1" : "mb-5"
                        }`}
                    >
                        {paymentLink?.merchantName}
                    </span>
                    <span className="block mx-auto text-xl font-medium text-card-desc text-center mb-8 sm:mb-3">
                        {paymentLink?.description || <i>No description provided</i>}
                    </span>
                    {isDonation ? (
                        <div className="mb-5">
                            <label className="block text-sm font-medium text-card-desc mb-2" htmlFor="donationAmount">
                                Donation amount
                            </label>
                            <div className="relative">
                                <input
                                    id="donationAmount"
                                    type="number"
                                    min="0.01"
                                    step="0.01"
                                    inputMode="decimal"
                                    className={`h-14 border rounded-xl w-full py-3 pl-4 pr-16 leading-tight font-medium text-2xl focus:outline-none focus:shadow-outline ${
                                        donationAmountError ? "border-main-error" : "border-main-green-3"
                                    }`}
                                    placeholder="0.00"
                                    value={donationAmount}
                                    onChange={(event) => {
                                        setDonationAmount(event.target.value);
                                        setDonationAmountError("");
                                    }}
                                />
                                <span className="absolute right-4 top-1/2 -translate-y-1/2 font-medium text-card-desc">
                                    {paymentLink.currency}
                                </span>
                            </div>
                            {donationAmountError && (
                                <span className="block mt-2 font-medium text-main-error text-xs">
                                    {donationAmountError}
                                </span>
                            )}
                        </div>
                    ) : (
                        <span className="block font-medium text-center text-3xl mb-4">
                            {renderCurrency(paymentLink.currency, paymentLink.price ?? undefined)}
                        </span>
                    )}
                    <button
                        className="relative border rounded-3xl bg-main-green-1 border-main-green-1 h-14 font-medium text-xl text-white flex items-center justify-center basis-full w-full"
                        onClick={() => createLink()}
                    >
                        {isDonation ? "Donate with crypto" : "Pay with crypto"}
                        <Icon name="arrow_right_white" className="absolute h-5 w-5 right-12 xs:right-5 md:right-6" />
                    </button>
                </div>
            ) : (
                <>
                    <LoadingTextIcon marginBottom={2} />
                    <LoadingTextIcon marginBottom={2} />
                    <LoadingTextIcon />
                </>
            )}
        </>
    );
};

export default LinkPage;
