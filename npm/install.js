#!/usr/bin/env node
/**
 * Post-install script that downloads the appropriate binary for the platform
 */

const fs = require('fs');
const path = require('path');
const https = require('https');
const { execSync } = require('child_process');

const BINARY_NAME = 'searxng-web-fetch-mcp';
const GITHUB_REPO = 'enrell/searxng-web-fetch-mcp';

const platformMap = {
  'linux': 'linux-x86_64',
  'darwin': 'darwin-x86_64',
  'win32': 'windows-x86_64',
};

const binDir = path.join(__dirname, 'bin');
const binPath = path.join(binDir, BINARY_NAME);

function getPlatform() {
  const platform = process.platform;
  const arch = process.arch;

  if (platform === 'linux' && arch === 'x64') {
    return 'linux-x86_64';
  }
  if (platform === 'darwin') {
    return arch === 'arm64' ? 'darwin-aarch64' : 'darwin-x86_64';
  }

  console.error(`Unsupported platform: ${platform}-${arch}`);
  console.error('Please build from source or use Docker.');
  process.exit(1);
}

async function downloadBinary(version, platform) {
  const url = `https://github.com/${GITHUB_REPO}/releases/download/v${version}/searxng-web-fetch-mcp-${platform}`;

  console.log(`Downloading ${BINARY_NAME} v${version} for ${platform}...`);

  return new Promise((resolve, reject) => {
    const request = https.get(url, {
      headers: {
        'User-Agent': 'searxng-web-fetch-mcp-installer'
      },
      redirect: 'follow'
    }, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        https.get(response.headers.location, (redirectResponse) => {
          handleResponse(redirectResponse, resolve, reject);
        }).on('error', reject);
      } else if (response.statusCode === 200) {
        handleResponse(response, resolve, reject);
      } else {
        reject(new Error(`HTTP ${response.statusCode}: ${url}`));
      }
    });

    request.on('error', reject);
    request.end();
  });
}

function handleResponse(response, resolve, reject) {
  const chunks = [];
  response.on('data', (chunk) => chunks.push(chunk));
  response.on('end', () => {
    const buffer = Buffer.concat(chunks);
    fs.mkdirSync(binDir, { recursive: true });
    fs.writeFileSync(binPath, buffer);
    fs.chmodSync(binPath, 0o755);
    console.log(`Binary installed to: ${binPath}`);
    resolve();
  });
  response.on('error', reject);
}

async function getLatestVersion() {
  return new Promise((resolve, reject) => {
    const request = https.get(`https://api.github.com/repos/${GITHUB_REPO}/releases/latest`, {
      headers: {
        'User-Agent': 'searxng-web-fetch-mcp-installer'
      }
    }, (response) => {
      let data = '';
      response.on('data', (chunk) => data += chunk);
      response.on('end', () => {
        try {
          const release = JSON.parse(data);
          resolve(release.tag_name.replace(/^v/, ''));
        } catch (e) {
          reject(new Error('Failed to parse release info'));
        }
      });
    });
    request.on('error', reject);
    request.end();
  });
}

async function main() {
  try {
    // Check if binary already exists (e.g., if user built from source)
    if (fs.existsSync(binPath)) {
      console.log('Binary already exists, skipping download.');
      return;
    }

    const version = require('../package.json').version;
    const platform = getPlatform();

    await downloadBinary(version, platform);
  } catch (error) {
    console.error('Failed to download binary:', error.message);
    console.error('\nYou can also build from source:');
    console.error('  git clone https://github.com/' + GITHUB_REPO + '.git');
    console.error('  cd searxng-web-fetch-mcp');
    console.error('  shards install --without development');
    console.error('  crystal build src/searxng_web_fetch_mcp.cr --release');
    process.exit(1);
  }
}

main();
