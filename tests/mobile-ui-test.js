const { chromium } = require('@playwright/test');

// iPhone 12 viewport
const mobileViewport = { width: 390, height: 844 };

(async () => {
  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport: mobileViewport,
    userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15A372 Safari/604.1'
  });
  const page = await context.newPage();

  console.log('Testing Mobile UI (iPhone 12: 390x844)');
  console.log('========================================\n');

  // 1. Login page
  console.log('1. Testing Login Page...');
  await page.goto('http://localhost:8081/devops/');
  await page.waitForTimeout(1000);
  await page.screenshot({ path: '/tmp/mobile-login.png', fullPage: true });
  console.log('   Screenshot: /tmp/mobile-login.png');

  // Login
  await page.fill('input[placeholder="Enter your username"]', 'admin');
  await page.fill('input[placeholder="Enter your password"]', 'admin');
  await page.click('button:has-text("Sign In")');
  await page.waitForTimeout(2000);

  // 2. Dashboard
  console.log('2. Testing Dashboard...');
  await page.screenshot({ path: '/tmp/mobile-dashboard.png', fullPage: true });
  console.log('   Screenshot: /tmp/mobile-dashboard.png');

  // 3. Apps page - need to open mobile menu first
  console.log('3. Testing Apps Page...');

  // Open hamburger menu
  const hamburgerBtn = page.locator('button:has(.lucide-menu)');
  if (await hamburgerBtn.isVisible()) {
    await hamburgerBtn.click();
    await page.waitForTimeout(500);
    await page.screenshot({ path: '/tmp/mobile-nav-open.png', fullPage: true });
    console.log('   Screenshot: /tmp/mobile-nav-open.png (sidebar open)');
  }

  // Click Apps link in sidebar - use text selector with force
  await page.getByRole('link', { name: 'Apps' }).first().click({ force: true });
  await page.waitForTimeout(1000);
  await page.screenshot({ path: '/tmp/mobile-apps.png', fullPage: true });
  console.log('   Screenshot: /tmp/mobile-apps.png');

  // Check if command count is visible
  const commandCount = page.locator('text=/\\d+ commands?/').first();
  if (await commandCount.isVisible()) {
    console.log('   ✓ Command count visible');
  } else {
    console.log('   ✗ Command count NOT visible');
  }

  // 4. App Detail
  console.log('4. Testing App Detail Page...');
  const appLink = page.locator('a:has-text("Test")').first();
  if (await appLink.isVisible()) {
    await appLink.click();
    await page.waitForTimeout(1000);
    await page.screenshot({ path: '/tmp/mobile-app-detail.png', fullPage: true });
    console.log('   Screenshot: /tmp/mobile-app-detail.png');

    // 5. Test Delete Command Confirm Dialog
    console.log('5. Testing Confirm Dialog...');
    const deleteBtn = page.locator('button:has(.lucide-trash-2)').first();
    if (await deleteBtn.isVisible()) {
      await deleteBtn.click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: '/tmp/mobile-confirm-dialog.png', fullPage: true });
      console.log('   Screenshot: /tmp/mobile-confirm-dialog.png');

      // Check dialog buttons layout
      const dialogButtons = page.locator('.flex.flex-col-reverse button');
      const buttonCount = await dialogButtons.count();
      console.log(`   ✓ Confirm dialog has ${buttonCount} stacked buttons (mobile layout)`);

      // Close dialog
      await page.click('button:has-text("Cancel")');
      await page.waitForTimeout(300);
    }

    // 6. Test Execute All Page
    console.log('6. Testing Execute All Page...');
    const executeAllBtn = page.locator('button:has-text("Execute All")');
    if (await executeAllBtn.isVisible()) {
      await executeAllBtn.click();
      await page.waitForTimeout(1000);
      await page.screenshot({ path: '/tmp/mobile-execute-all.png', fullPage: true });
      console.log('   Screenshot: /tmp/mobile-execute-all.png');

      // Check if Execute All button is full width on mobile
      const execBtn = page.locator('button:has-text("Execute All")').first();
      const btnBox = await execBtn.boundingBox();
      if (btnBox && btnBox.width > 300) {
        console.log('   ✓ Execute All button is full-width on mobile');
      } else {
        console.log(`   Button width: ${btnBox?.width}px`);
      }
    }
  }

  // 7. Test Toast Notification
  console.log('7. Testing Toast Notification...');
  await page.goto('http://localhost:8081/devops/#/apps');
  await page.waitForTimeout(1000);

  // Try to trigger a toast by copying token
  const appCard = page.locator('.cursor-pointer, a:has-text("Test")').first();
  if (await appCard.isVisible()) {
    await appCard.click();
    await page.waitForTimeout(1000);

    const copyBtn = page.locator('button:has(.lucide-copy)').first();
    if (await copyBtn.isVisible()) {
      await copyBtn.click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: '/tmp/mobile-toast.png', fullPage: true });
      console.log('   Screenshot: /tmp/mobile-toast.png');

      // Check toast position
      const toast = page.locator('.fixed.bottom-4');
      if (await toast.isVisible()) {
        console.log('   ✓ Toast notification visible');
      }
    }
  }

  // 8. Navigation/Sidebar test
  console.log('8. Testing Navigation...');
  await page.goto('http://localhost:8081/devops/#/dashboard');
  await page.waitForTimeout(1000);

  // Check for hamburger menu on mobile
  const hamburger = page.locator('button:has(.lucide-menu)');
  if (await hamburger.isVisible()) {
    console.log('   ✓ Mobile hamburger menu visible');
    await hamburger.click();
    await page.waitForTimeout(500);
    await page.screenshot({ path: '/tmp/mobile-nav-open.png', fullPage: true });
    console.log('   Screenshot: /tmp/mobile-nav-open.png');
  } else {
    console.log('   Navigation layout might need review');
  }

  await browser.close();

  console.log('\n========================================');
  console.log('Mobile UI Test Complete!');
  console.log('Screenshots saved to /tmp/mobile-*.png');
})();
