const { chromium } = require('@playwright/test');

(async () => {
  const browser = await chromium.launch();

  // Desktop view
  const desktopContext = await browser.newContext({
    viewport: { width: 1920, height: 1080 }
  });
  const desktopPage = await desktopContext.newPage();

  console.log('Capturing Modern Login Page Screenshots...');
  console.log('==========================================\n');

  // After - Desktop
  console.log('1. Desktop Login (After - Modern)');
  await desktopPage.goto('http://localhost:8081/devops/');
  await desktopPage.waitForTimeout(1500);
  await desktopPage.screenshot({ path: '/tmp/login-after-desktop.png', fullPage: true });
  console.log('   Screenshot: /tmp/login-after-desktop.png');

  // Mobile view
  const mobileContext = await browser.newContext({
    viewport: { width: 390, height: 844 },
    userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)'
  });
  const mobilePage = await mobileContext.newPage();

  // After - Mobile
  console.log('2. Mobile Login (After - Modern)');
  await mobilePage.goto('http://localhost:8081/devops/');
  await mobilePage.waitForTimeout(1500);
  await mobilePage.screenshot({ path: '/tmp/login-after-mobile.png', fullPage: true });
  console.log('   Screenshot: /tmp/login-after-mobile.png');

  await browser.close();

  console.log('\n==========================================');
  console.log('Screenshots saved! Compare before vs after:');
  console.log('  Before Desktop: /tmp/login-before-desktop.png');
  console.log('  After Desktop:  /tmp/login-after-desktop.png');
  console.log('  Before Mobile:  /tmp/login-before-mobile.png');
  console.log('  After Mobile:   /tmp/login-after-mobile.png');
})();
