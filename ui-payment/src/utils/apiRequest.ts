import axios, {AxiosError} from "axios";
import {RenderErrorAlert} from "src/components/ErrorAlert";

const apiRequest = axios.create({
    baseURL: import.meta.env.VITE_BACKEND_HOST,
    headers: {
        "Content-Type": "application/json",
        "Cache-Control": "no-cache",
        Accept: "application/json"
    },
    withCredentials: true
});

const isNetworkError = (error: unknown) => axios.isAxiosError(error) && !error.response;

const renderErrorMessage = (error: Error | AxiosError) => {
    if (isNetworkError(error)) {
        return "Network error. Invoice will keep retrying.";
    }

    return error.message;
};

apiRequest.interceptors.response.use(undefined, function (error: Error | AxiosError) {
    RenderErrorAlert(renderErrorMessage(error));
    return Promise.reject(error);
});

export default apiRequest;
export {isNetworkError};
