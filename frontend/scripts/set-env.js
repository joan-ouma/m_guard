const fs = require('fs');
const path = require('path');

const isProd = process.argv[2] === 'production' || process.env.PRODUCTION === 'true';
const targetFileName = isProd ? 'environment.prod.ts' : 'environment.ts';
const targetPath = path.resolve(__dirname, '../src/environments', targetFileName);

fs.mkdirSync(path.dirname(targetPath), { recursive: true });

const fileContent = `export const environment = {
  production: ${isProd},
  apiBaseUrl: '${process.env.API_BASE_URL || ''}',
};
`;

fs.writeFileSync(targetPath, fileContent, 'utf-8');
