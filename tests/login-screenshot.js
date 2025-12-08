const { chromium } = require('@playwright/test');

(async () => {
  const browser = await chromium.launch();

  // Desktop view
  const desktopContext = await browser.newContext({
    viewport: { width: 1920, height: 1080 }
  });
  const desktopPage = await desktopContext.newPage();

  console.log('Capturing Login Page Screenshots...');
  console.log('=====================================\n');

  // Before - Desktop
  console.log('1. Desktop Login (Before)');
  await desktopPage.goto('http://localhost:8081/devops/');
  await desktopPage.waitForTimeout(1000);
  await desktopPage.screenshot({ path: '/tmp/login-before-desktop.png', fullPage: true });
  console.log('   Screenshot: /tmp/login-before-desktop.png');

  // Mobile view
  const mobileContext = await browser.newContext({
    viewport: { width: 390, height: 844 },
    userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)'
  });
  const mobilePage = await mobileContext.newPage();

  // Before - Mobile
  console.log('2. Mobile Login (Before)');
  await mobilePage.goto('http://localhost:8081/devops/');
  await mobilePage.waitForTimeout(1000);
  await mobilePage.screenshot({ path: '/tmp/login-before-mobile.png', fullPage: true });
  console.log('   Screenshot: /tmp/login-before-mobile.png');

  await browser.close();

  console.log('\n=====================================');
  console.log('Screenshots saved!');
})();
