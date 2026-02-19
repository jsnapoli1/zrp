import fs from 'fs';

async function globalSetup() {
  const testDir = '/tmp/zrp-test';
  const testDbPath = '/tmp/zrp-test/zrp-test.db';
  const testPartsDir = '/tmp/zrp-test/parts';

  console.log('Setting up test environment...');
  console.log('Test directory:', testDir);

  // Create test directories
  if (!fs.existsSync(testDir)) {
    fs.mkdirSync(testDir, { recursive: true });
    console.log('Created test directory');
  }
  if (!fs.existsSync(testPartsDir)) {
    fs.mkdirSync(testPartsDir, { recursive: true });
    console.log('Created parts directory');
  }

  // Clean up old test database
  if (fs.existsSync(testDbPath)) {
    fs.unlinkSync(testDbPath);
    console.log('Removed old test database');
  }
  
  console.log('Test environment setup complete');
}

export default globalSetup;
