import { appTools, defineConfig } from "@modern-js/app-tools";

const devServerPort = process.env.PORT ? Number.parseInt(process.env.PORT, 10) : undefined;

export default defineConfig({
  html: {
    outputStructure: "flat",
    title: "Monorepo Template Demo"
  },
  output: {
    distPath: {
      html: ""
    }
  },
  ...(devServerPort ? { server: { port: devServerPort } } : {}),
  plugins: [appTools()]
});
