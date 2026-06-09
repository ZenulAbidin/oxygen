import {EthereumProvider} from "@walletconnect/ethereum-provider";
import type {PaymentInfo, PaymentMethod} from "src/types";

const EVM_BLOCKCHAINS = new Set(["ETH", "MATIC", "BSC"]);
const ERC20_TRANSFER_SELECTOR = "a9059cbb";

interface SendWalletConnectPaymentParams {
    paymentMethod: PaymentMethod;
    paymentInfo: PaymentInfo;
    projectId: string;
}

interface WalletConnectPaymentResult {
    fromAddress: string;
    transactionHash: string;
}

type WalletConnectProvider = Awaited<ReturnType<typeof EthereumProvider.init>>;

export function canPayWithWalletConnect(paymentMethod?: PaymentMethod) {
    if (!paymentMethod || !EVM_BLOCKCHAINS.has(paymentMethod.blockchain)) {
        return false;
    }

    if (paymentMethod.currencyType === "coin") {
        return true;
    }

    return paymentMethod.currencyType === "token" && Boolean(paymentMethod.tokenContractAddress);
}

export async function sendWalletConnectPayment({
    paymentMethod,
    paymentInfo,
    projectId
}: SendWalletConnectPaymentParams): Promise<WalletConnectPaymentResult> {
    const chainId = parseChainId(paymentMethod);
    const provider = await initProvider(projectId, chainId);
    const fromAddress = await connectWallet(provider, chainId);
    const transaction = buildTransaction(paymentMethod, paymentInfo, fromAddress);

    const transactionHash = await provider.request<string>({
        method: "eth_sendTransaction",
        params: [transaction]
    });

    return {fromAddress, transactionHash};
}

async function initProvider(projectId: string, chainId: number) {
    return EthereumProvider.init({
        projectId,
        optionalChains: [chainId],
        optionalMethods: ["eth_requestAccounts", "eth_sendTransaction", "wallet_switchEthereumChain"],
        optionalEvents: ["accountsChanged", "chainChanged", "connect", "disconnect"],
        showQrModal: true,
        metadata: {
            name: "Oxygen",
            description: "Oxygen payment checkout",
            url: window.location.origin,
            icons: []
        }
    });
}

async function connectWallet(provider: WalletConnectProvider, chainId: number) {
    let accounts = provider.connected ? provider.accounts : [];
    if (accounts.length === 0) {
        accounts = await provider.enable();
    }

    const fromAddress = normalizeAccount(accounts[0]);
    await switchChain(provider, chainId);

    return fromAddress;
}

async function switchChain(provider: WalletConnectProvider, chainId: number) {
    if (provider.chainId === chainId) {
        return;
    }

    try {
        await provider.request({
            method: "wallet_switchEthereumChain",
            params: [{chainId: toHexQuantity(String(chainId))}]
        });
    } catch {
        if (provider.chainId !== chainId) {
            throw new Error("Connected wallet is not on the selected payment network.");
        }
    }
}

function buildTransaction(paymentMethod: PaymentMethod, paymentInfo: PaymentInfo, fromAddress: string) {
    if (paymentMethod.currencyType === "coin") {
        return {
            from: fromAddress,
            to: normalizeAddress(paymentInfo.recipientAddress),
            value: toHexQuantity(paymentInfo.amount)
        };
    }

    if (paymentMethod.currencyType === "token" && paymentMethod.tokenContractAddress) {
        return {
            from: fromAddress,
            to: normalizeAddress(paymentMethod.tokenContractAddress),
            value: "0x0",
            data: encodeERC20Transfer(paymentInfo.recipientAddress, paymentInfo.amount)
        };
    }

    throw new Error("WalletConnect is not available for the selected payment method.");
}

function encodeERC20Transfer(recipientAddress: string, amountRaw: string) {
    const recipient = normalizeAddress(recipientAddress).slice(2).padStart(64, "0");
    const amount = uint256Hex(amountRaw);

    return `0x${ERC20_TRANSFER_SELECTOR}${recipient}${amount}`;
}

function normalizeAccount(account?: string) {
    if (!account) {
        throw new Error("Wallet did not provide an account.");
    }

    const parts = account.split(":");
    return normalizeAddress(parts[parts.length - 1]);
}

function normalizeAddress(address: string) {
    if (!/^0x[0-9a-fA-F]{40}$/.test(address)) {
        throw new Error("Selected payment method does not have a valid EVM address.");
    }

    return address;
}

function parseChainId(paymentMethod: PaymentMethod) {
    const chainId = Number(paymentMethod.networkId);
    if (!Number.isSafeInteger(chainId) || chainId <= 0) {
        throw new Error("Selected payment method does not have a valid EVM chain ID.");
    }

    return chainId;
}

function toHexQuantity(value: string) {
    const parsed = BigInt(value);
    if (parsed < 0n) {
        throw new Error("Invalid negative EVM quantity.");
    }

    return `0x${parsed.toString(16)}`;
}

function uint256Hex(value: string) {
    const parsed = BigInt(value);
    if (parsed < 0n) {
        throw new Error("Invalid negative token amount.");
    }

    const hex = parsed.toString(16);
    if (hex.length > 64) {
        throw new Error("Token amount exceeds uint256.");
    }

    return hex.padStart(64, "0");
}
