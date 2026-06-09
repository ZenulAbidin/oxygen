import path from "path";
import {defineConfig} from "vite";
import react from "@vitejs/plugin-react";
import dynamicImport from "vite-plugin-dynamic-import";
import basicSsl from "@vitejs/plugin-basic-ssl";
import svgr from "vite-plugin-svgr";

const rootPath = process.env.VITE_ROOTPATH ?? "/dashboard/";

// https://vitejs.dev/config/
export default defineConfig({
    base: rootPath,
    build: {
        sourcemap: false,
        minify: "esbuild"
    },
    resolve: {
        alias: {
            src: path.resolve(__dirname, "/src")
        }
    },
    plugins: [basicSsl(), svgr(), dynamicImport(), react()]
});
