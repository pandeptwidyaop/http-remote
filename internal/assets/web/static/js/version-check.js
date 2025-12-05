// Check for updates and show banner if available
(function() {
  const pathPrefix = window.pathPrefix || '';
  
  // Check for updates on page load
  async function checkForUpdates() {
    try {
      const response = await fetch(`${pathPrefix}/api/version/check`);
      if (!response.ok) return;
      
      const data = await response.json();
      
      if (data.update_available) {
        showUpdateBanner(data.current, data.latest, data.release_name);
      }
    } catch (error) {
      // Silently fail - don't disturb user if check fails
      console.debug('Version check failed:', error);
    }
  }
  
  function showUpdateBanner(current, latest, releaseName) {
    // Don't show banner if already dismissed
    const dismissed = sessionStorage.getItem('update-banner-dismissed');
    if (dismissed === latest) return;
    
    const banner = document.createElement('div');
    banner.className = 'update-banner';
    banner.innerHTML = `
      <div class="update-banner-content">
        <div class="update-banner-icon">🎉</div>
        <div class="update-banner-text">
          <strong>New version available!</strong>
          <span>Version ${latest} is now available (you're on ${current})</span>
        </div>
        <div class="update-banner-actions">
          <a href="https://github.com/pandeptwidyaop/http-remote/releases/latest" 
             target="_blank" 
             class="btn btn-sm">
            View Release
          </a>
          <button onclick="dismissUpdateBanner('${latest}')" class="btn btn-sm">
            Dismiss
          </button>
        </div>
      </div>
    `;
    
    document.body.insertBefore(banner, document.body.firstChild);
    
    // Add animation
    setTimeout(() => banner.classList.add('show'), 10);
  }
  
  window.dismissUpdateBanner = function(version) {
    sessionStorage.setItem('update-banner-dismissed', version);
    const banner = document.querySelector('.update-banner');
    if (banner) {
      banner.classList.remove('show');
      setTimeout(() => banner.remove(), 300);
    }
  };
  
  // Check on page load
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', checkForUpdates);
  } else {
    checkForUpdates();
  }
})();
