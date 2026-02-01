import type { Plugin } from "@opencode-ai/plugin"

export const FixAfterEditPlugin: Plugin = async ({ $ }) => {
  return {
    "tool.execute.after": async (input, output) => {
      // Check if this was a file modification tool
      if (input.tool === "edit" || input.tool === "write" || input.tool === "patch") {
        // Extract filePath from output.args (based on OpenCode plugin API)
        const filePath = output?.args?.filePath

        if (filePath) {
          try {
            // Run just fix on the modified file(s)
            await $`just fix ${filePath}`
          } catch {
            // Silently ignore errors
          }
        }
      }
    },
  }
}
