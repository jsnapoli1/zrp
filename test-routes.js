const { chromium } = require('playwright');

const routes = [
  '/dashboard',
  '/parts',
  '/ecos',
  '/documents',
  '/inventory',
  '/procurement',
  '/vendors',
  '/work-orders',
  '/ncrs',
  '/rmas',
  '/testing',
  '/devices',
  '/firmware',
  '/quotes',
  '/calendar',
  '/reports',
  '/audit',
  '/users',
  '/api-keys',
  '/email-settings'
];

async function testRoutes() {
  console.log('🚀 Starting React route tests...\n');
  
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  
  // Set a longer timeout
  page.setDefaultTimeout(10000);
  
  const results = [];
  
  for (const route of routes) {
    try {
      console.log(`Testing ${route}...`);
      
      await page.goto(`http://localhost:5173${route}`, { waitUntil: 'networkidle' });
      
      // Wait for React to render
      await page.waitForTimeout(2000);
      
      // Check if there's an error boundary or white screen
      const hasError = await page.evaluate(() => {
        // Check for common error indicators
        const body = document.body.innerText.toLowerCase();
        return body.includes('something went wrong') || 
               body.includes('error') || 
               body.includes('failed to compile') ||
               document.body.innerHTML.trim().length === 0;
      });
      
      // Check if the page has actual content
      const hasContent = await page.evaluate(() => {
        return document.body.innerText.trim().length > 0;
      });
      
      const title = await page.title();
      
      if (hasError) {
        results.push({ route, status: '❌ ERROR', details: 'Error detected in page' });
        console.log(`  ❌ ${route} - Error detected`);
      } else if (!hasContent) {
        results.push({ route, status: '⚠️ EMPTY', details: 'No content rendered' });
        console.log(`  ⚠️ ${route} - No content`);
      } else {
        results.push({ route, status: '✅ OK', details: `Title: ${title}` });
        console.log(`  ✅ ${route} - OK`);
      }
      
    } catch (error) {
      results.push({ route, status: '❌ FAILED', details: error.message });
      console.log(`  ❌ ${route} - Failed: ${error.message}`);
    }
  }
  
  await browser.close();
  
  // Summary
  console.log('\n📊 SUMMARY:');
  console.log('============');
  
  const ok = results.filter(r => r.status === '✅ OK').length;
  const errors = results.filter(r => r.status.includes('❌')).length;
  const warnings = results.filter(r => r.status.includes('⚠️')).length;
  
  console.log(`✅ Working: ${ok}`);
  console.log(`❌ Broken: ${errors}`);  
  console.log(`⚠️ Empty: ${warnings}`);
  
  if (errors > 0) {
    console.log('\n🔥 BROKEN PAGES:');
    results.filter(r => r.status.includes('❌')).forEach(r => {
      console.log(`  ${r.route}: ${r.details}`);
    });
  }
  
  console.log(`\n🎯 Overall: ${errors === 0 ? '✅ All routes working!' : '⚠️ Some issues found'}`);
}

testRoutes().catch(console.error);