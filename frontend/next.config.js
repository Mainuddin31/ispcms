/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  env: {
    // Keep client calls same-origin and let Next rewrite proxy to backend.
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || '/api/v1',
  },
  async rewrites() {
    const apiProxyTarget = process.env.API_PROXY_TARGET || 'http://backend:8080';
    return [
      {
        source: '/api/v1/:path*',
        destination: `${apiProxyTarget}/api/v1/:path*`,
      },
    ];
  },
}

module.exports = nextConfig
