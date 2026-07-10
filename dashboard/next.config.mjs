/** @type {import('next').NextConfig} */
const nextConfig = {
  // Static export so the whole dashboard is plain files GitHub Pages can serve.
  output: "export",
  // Project Pages live under /<repo>/. The workflow sets PAGES_BASE_PATH; a
  // custom domain (served at root) just sets it to "".
  basePath: process.env.PAGES_BASE_PATH || "",
  assetPrefix: process.env.PAGES_BASE_PATH || undefined,
  images: { unoptimized: true },
  trailingSlash: true,
};

export default nextConfig;
