/** @type {import('next').NextConfig} */
const nextConfig = {
  // Every page reads the live read API; nothing is prerendered at build time, so
  // `pnpm -r build` stays green with no stack running (and CI never needs the API).
  reactStrictMode: true,
};

export default nextConfig;
