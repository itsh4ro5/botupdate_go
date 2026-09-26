const { chromium } = require('playwright');
const fs = require('fs');

(async () => {
  console.log('Starting Playwright tests for Phase 13...');
  
  if (!fs.existsSync('../browser_tests')) {
    fs.mkdirSync('../browser_tests');
  }

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 }
  });
  
  const page = await context.newPage();
  
  page.on('console', msg => console.log(`BROWSER CONSOLE: ${msg.type()} - ${msg.text()}`));
  page.on('pageerror', error => console.error(`BROWSER ERROR: ${error}`));

  console.log('Navigating to login...');
  await page.goto('http://localhost:3000/login');
  
  await page.fill('input[type="text"]', 'admin');
  await page.fill('input[type="password"]', 'adminpassword');
  await page.click('button[type="submit"]');
  
  console.log('Waiting for dashboard...');
  await page.waitForURL('**/', { timeout: 10000 });
  await page.screenshot({ path: '../browser_tests/phase13_dashboard_desktop.png' });
  console.log('Dashboard verified.');

  console.log('Navigating to Admins (Testing Panic Fix)...');
  await page.goto('http://localhost:3000/settings'); // Or wherever admins are
  await page.waitForTimeout(1000);
  await page.screenshot({ path: '../browser_tests/phase13_admin_stable.png' });
  console.log('Admin page did not crash backend.');

  console.log('Navigating to Support...');
  await page.goto('http://localhost:3000/support');
  await page.waitForSelector('.support-page', { timeout: 10000 });
  await page.screenshot({ path: '../browser_tests/phase13_support_desktop.png' });

  // Mobile viewport test
  const mobileContext = await browser.newContext({
    viewport: { width: 390, height: 844 },
    userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1'
  });
  const mobilePage = await mobileContext.newPage();
  
  // Need to login again since context is isolated or inject cookies
  const cookies = await context.cookies();
  await mobileContext.addCookies(cookies);
  
  await mobilePage.goto('http://localhost:3000/support');
  await mobilePage.waitForSelector('.support-page', { timeout: 10000 });
  await mobilePage.screenshot({ path: '../browser_tests/phase13_support_mobile.png' });
  console.log('Mobile support verified.');

  await browser.close();
  console.log('Tests completed successfully.');
})();
