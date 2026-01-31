// Build script to process HTML includes
import posthtml from 'posthtml';
import include from 'posthtml-include';
import { readFileSync, writeFileSync, mkdirSync } from 'fs';
import { dirname } from 'path';

const srcPath = './src/index.html';
const distPath = './dist/index.html';

// Ensure dist directory exists
mkdirSync(dirname(distPath), { recursive: true });

// Read source HTML
const html = readFileSync(srcPath, 'utf8');

// Process with PostHTML (await the promise)
try {
  const result = await posthtml([
    include({ root: './src' })
  ]).process(html);

  writeFileSync(distPath, result.html);
  console.log('HTML processed successfully');
} catch (err) {
  console.error('HTML processing failed:', err);
  process.exit(1);
}
