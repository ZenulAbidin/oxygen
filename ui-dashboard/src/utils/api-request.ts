import axios, {AxiosError, AxiosRequestConfig} from "axios";
import {notification} from "antd";
import {ErrorResponse} from "src/types";
import withApiPath from "src/utils/with-api-path";

export const DASHBOARD_UNAUTHORIZED_EVENT = "oxygen-dashboard-unauthorized";

interface RetryableAxiosRequestConfig extends AxiosRequestConfig {
    _retry?: boolean;
}

const apiRequest = axios.create({
    baseURL: import.meta.env.VITE_BACKEND_HOST,
    headers: {
        "Content-Type": "application/json",
        "Cache-Control": "no-cache",
        Accept: "application/json"
    },
    withCredentials: true
});

let unauthorizedNotificationShown = false;

export const setDashboardCSRFToken = (csrf?: string) => {
    if (csrf) {
        apiRequest.defaults.headers.common["x-csrf-token"] = csrf;
        unauthorizedNotificationShown = false;
    }
};

const refreshCSRFToken = async (): Promise<string> => {
    const response = await apiRequest.get(withApiPath(`/auth/csrf-cookie`));
    const csrf = response.headers["x-csrf-token"];

    setDashboardCSRFToken(csrf);

    return csrf ?? "";
};

const dispatchUnauthorized = () => {
    if (!unauthorizedNotificationShown) {
        unauthorizedNotificationShown = true;
        notification.error({
            message: "Authentication required",
            description: "Please sign in again.",
            placement: "bottomRight"
        });
    }

    window.dispatchEvent(new Event(DASHBOARD_UNAUTHORIZED_EVENT));
};

apiRequest.interceptors.response.use(undefined, async (error: AxiosError) => {
    if (!error.response) {
        return Promise.reject(error);
    }

    if (error.response.status === 400) {
        const response: ErrorResponse = error.response.data as ErrorResponse;

        if (!response?.errors) {
            return;
        }

        const errors = response.errors.length
            ? response.errors
                  .map((item) => {
                      return item.message;
                  })
                  .join(", ")
            : "";

        if (response.status === "validation_error") {
            notification.error({
                message: response.message,
                description: !errors ? "Validation error" : "Validation error: " + errors + ".",
                placement: "bottomRight"
            });
        } else {
            notification.error({
                message: response.message,
                description: !errors ? "Validation error" : "Got the following errors: " + errors + ".",
                placement: "bottomRight"
            });
        }
    } else if (error.response.status === 403) {
        const originalRequest = error.config as RetryableAxiosRequestConfig | undefined;

        if (!originalRequest || originalRequest._retry) {
            return Promise.reject(error);
        }

        originalRequest._retry = true;

        try {
            const newToken = await refreshCSRFToken();

            if (originalRequest.headers && newToken.length) {
                originalRequest.headers["x-csrf-token"] = newToken;
                return apiRequest.request(originalRequest);
            }
        } catch (e) {
            console.error("Ocurred a error: ", e);
        }
    } else if (error.response.status === 401) {
        dispatchUnauthorized();
    } else if (error.response.status !== 401) {
        notification.error({
            message: "Something went wrong",
            description: error.message,
            placement: "bottomRight"
        });
    }

    return Promise.reject(error);
});

export default apiRequest;
