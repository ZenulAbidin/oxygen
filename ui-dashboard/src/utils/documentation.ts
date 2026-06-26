const configuredDocumentationURL = import.meta.env.VITE_DOCUMENTATION_URL as string | undefined;
const documentationBaseURL = (configuredDocumentationURL?.trim() || "/docs/http").replace(/\/+$/, "");

const documentationURLs = {
    home: documentationBaseURL,
    merchantAPI: `${documentationBaseURL}/api/merchant.html`,
    webhooks: `${documentationBaseURL}/#webhooks`
};

export default documentationURLs;
