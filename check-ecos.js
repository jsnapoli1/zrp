const { chromium } = require('@playwright/test');
(async () => {
  const b = await chromium.launch();
  const p = await b.newPage();
  p.on('console', msg => console.log('BROWSER:', msg.text()));
  await p.goto('http://localhost:9000');
  await p.waitForSelector('#login-page:not(.hidden)', {timeout:5000});
  await p.fill('#login-username','admin');
  await p.fill('#login-password','changeme');
  await p.click('#login-form button[type="submit"]');
  await p.waitForSelector('#app:not(.hidden)', {timeout:5000});
  await p.evaluate(()=>{localStorage.setItem('zrp-tour-seen','true');document.querySelectorAll('.zt-overlay-bg,.zt-overlay,.zt-popover').forEach(e=>e.remove());});
  await p.waitForTimeout(1000);
  
  // Check hash before
  const hashBefore = await p.evaluate(() => window.location.hash);
  console.log('HASH BEFORE:', hashBefore);
  
  await p.evaluate(()=>window.navigate('ecos'));
  await p.waitForTimeout(500);
  
  const hashAfter = await p.evaluate(() => window.location.hash);
  console.log('HASH AFTER:', hashAfter);
  
  await p.waitForTimeout(3000);
  
  const hashFinal = await p.evaluate(() => window.location.hash);
  console.log('HASH FINAL:', hashFinal);
  
  const title = await p.evaluate(() => document.getElementById('page-title').textContent);
  console.log('PAGE TITLE:', title);
  
  const contentSnippet = await p.evaluate(() => document.getElementById('content').textContent.substring(0, 200));
  console.log('CONTENT:', contentSnippet);
  
  await b.close();
})();
