const { chromium } = require('playwright');
const path = require('path');
const fs = require('fs');

const OUT_DIR = 'C:\\Users\\kamal\\.gemini\\antigravity-ide\\brain\\b0b2c030-2834-47f5-9835-eecb6817bd02\\browser_tests';
if (!fs.existsSync(OUT_DIR)) {
  fs.mkdirSync(OUT_DIR, { recursive: true });
}

(async () => {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 }
  });
  const page = await context.newPage();

  page.on('console', msg => {
    if (msg.type() === 'error' || msg.type() === 'warning') {
      console.log(`BROWSER CONSOLE [${msg.type()}]:`, msg.text());
    }
  });
  
  page.on('pageerror', err => {
    console.log('BROWSER EXCEPTION:', err.message);
  });

  page.on('requestfailed', request => {
    console.log(`NETWORK FAILED: ${request.url()} - ${request.failure().errorText}`);
  });

  try {
    console.log('Navigating to login...');
    await page.goto('http://localhost:5173/login');
    await page.waitForLoadState('networkidle');
    await page.screenshot({ path: path.join(OUT_DIR, '01_login_desktop.png') });
    
    console.log('Logging in...');
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'adminpassword123');
    await page.click('button[type="submit"]');
    
    console.log('Checking for password change...');
    await page.waitForTimeout(2000);
    const url = page.url();
    if (url.includes('change-password')) {
      console.log('Password change required. Changing password...');
      // Fill current password
      await page.fill('input[placeholder="Current Password"]', 'adminpassword');
      // Fill new password
      await page.fill('input[placeholder="New Password (min 10 chars)"]', 'adminpassword123');
      // Confirm new password
      await page.fill('input[placeholder="Confirm New Password"]', 'adminpassword123');
      await page.click('button[type="submit"]');
      await page.waitForTimeout(2000);
    }
    
    console.log('Waiting for dashboard redirect...');
    await page.waitForURL('http://localhost:5173/');
    await page.waitForLoadState('networkidle');
    
    // Give time for WebSocket to connect and display SYSTEM ONLINE
    await page.waitForTimeout(2000);
    await page.screenshot({ path: path.join(OUT_DIR, '02_dashboard_desktop.png') });
    
    console.log('Testing Command Palette (Ctrl+K)...');
    await page.keyboard.press('Control+K');
    await page.waitForTimeout(500);
    await page.screenshot({ path: path.join(OUT_DIR, '03_command_palette.png') });
    await page.keyboard.press('Escape');
    
    console.log('Navigating to Users...');
    await page.goto('http://localhost:5173/users');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000); // let data load
    await page.screenshot({ path: path.join(OUT_DIR, '04_users_desktop.png') });
    
    console.log('Opening User Profile Modal...');
    // Click the first row in tbody
    const firstRow = await page.$('.users-table tbody tr');
    if (firstRow) {
      await firstRow.click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: path.join(OUT_DIR, '05_user_profile_modal.png') });
      await page.keyboard.press('Escape');
    } else {
      console.log('No users found in table.');
    }
    
    console.log('Testing Mobile View...');
    const mobileContext = await browser.newContext({ viewport: { width: 390, height: 844 } });
    const mobilePage = await mobileContext.newPage();
    
    // Auth relies on HttpOnly cookie, so mobileContext won't have it unless we share context or log in again.
    // Instead of new context, let's just resize the existing page
    await page.setViewportSize({ width: 390, height: 844 });
    await page.waitForTimeout(500);
    await page.screenshot({ path: path.join(OUT_DIR, '06_users_mobile.png') });
    
    await page.goto('http://localhost:5173/');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);
    await page.screenshot({ path: path.join(OUT_DIR, '07_dashboard_mobile.png') });

    console.log('Browser tests completed successfully.');
  } catch (err) {
    console.error('Test execution failed:', err);
  } finally {
    await browser.close();
  }
})();
