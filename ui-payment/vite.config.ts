import path from "path";
import {defineConfig} from "vite";
import react from "@vitejs/plugin-react";
import svgr from "vite-plugin-svgr";
import dynamicImport from "vite-plugin-dynamic-import";
import basicSsl from "@vitejs/plugin-basic-ssl";

const rootPath = process.env.VITE_ROOTPATH ?? "/p/";

// https://vitejs.dev/config/
export default defineConfig({
    base: rootPath,
    build: {
        target: "es2022"
    },
    resolve: {
        alias: {
            src: path.resolve(__dirname, "/src")
        }
    },
    // @ts-ignore
    plugins: [basicSsl(), svgr(), dynamicImport(), react()]
});
