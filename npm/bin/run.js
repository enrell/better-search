#!/usr/bin/env node
/**
 * Wrapper script that runs the binary
 */

const path = require('path');
const { spawn } = require('child_process');

const binaryPath = path.join(__dirname, '..' , 'native', 'searxng-web-fetch-mcp' + (process.platform === 'win32' ? '.exe' : ''));

const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  env: process.env
});

child.on('close', (code) => {
  process.exit(code);
});

child.on('error', (err) => {
  if (err.code === 'ENOENT') {
    console.error('Binary not found. Please run: npm install');
  } else {
    console.error('Failed to run binary:', err.message);
  }
  process.exit(1);
});
