const { chromium } = require('@playwright/test');

(async () => {
  const browser = await chromium.launch();
  const context = await browser.newContext();
  const page = await context.newPage();

  // Go to login page
  await page.goto('http://localhost:8081/devops/');

  // Login
  await page.fill('input[placeholder="Enter your username"]', 'admin');
  await page.fill('input[placeholder="Enter your password"]', 'admin');
  await page.click('button:has-text("Sign In")');

  // Wait for navigation
  await page.waitForTimeout(2000);

  // Go to Apps
  await page.click('a:has-text("Apps")');
  await page.waitForTimeout(1000);

  // Click on Test app
  await page.click('text=Test app');
  await page.waitForTimeout(1000);
  await page.screenshot({ path: '/tmp/app-detail.png', fullPage: true });
  console.log('App detail screenshot saved');

  // Check current command count
  const commandCount = await page.locator('.cursor-grab').count();
  console.log('Current command count:', commandCount);

  if (commandCount < 2) {
    // Add first command
    await page.click('button:has-text("New Command")');
    await page.waitForTimeout(500);
    await page.fill('input[placeholder="e.g., deploy"]', 'Echo Test 1');
    await page.fill('textarea[placeholder*="git pull"]', 'echo "Hello from command 1"');
    await page.click('button:has-text("Create Command")');
    await page.waitForTimeout(1000);

    // Add second command
    await page.click('button:has-text("New Command")');
    await page.waitForTimeout(500);
    await page.fill('input[placeholder="e.g., deploy"]', 'Echo Test 2');
    await page.fill('textarea[placeholder*="git pull"]', 'echo "Hello from command 2"');
    await page.click('button:has-text("Create Command")');
    await page.waitForTimeout(1000);

    // Add third command
    await page.click('button:has-text("New Command")');
    await page.waitForTimeout(500);
    await page.fill('input[placeholder="e.g., deploy"]', 'Echo Test 3');
    await page.fill('textarea[placeholder*="git pull"]', 'echo "Hello from command 3"');
    await page.click('button:has-text("Create Command")');
    await page.waitForTimeout(1000);
  }

  // Screenshot with commands
  await page.screenshot({ path: '/tmp/app-with-commands.png', fullPage: true });
  console.log('App with commands screenshot saved');

  // Click Execute All if visible
  const executeAllBtn = page.locator('button:has-text("Execute All")');
  if (await executeAllBtn.isVisible()) {
    await executeAllBtn.click();
    await page.waitForTimeout(1000);
    await page.screenshot({ path: '/tmp/execute-all.png', fullPage: true });
    console.log('Execute All page screenshot saved');
  } else {
    console.log('Execute All button not visible (need at least 2 commands)');
  }

  await browser.close();
})();
