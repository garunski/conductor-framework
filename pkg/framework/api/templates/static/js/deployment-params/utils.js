// Utility functions for deployment parameters

(function() {
    'use strict';
    
    window.DeploymentParams = window.DeploymentParams || {};
    
    // Use functions from scripts-core.html if available, otherwise define fallbacks
    DeploymentParams.Utils = {
        escapeHtml: function(text) {
            if (typeof window.escapeHtml === 'function') {
                return window.escapeHtml(text);
            }
            if (!text) return '';
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        },
        
        showStatus: function(message, type) {
            const statusDiv = document.getElementById('parameters-status');
            if (!statusDiv) return;
            
            statusDiv.textContent = '';
            statusDiv.className = 'small';
            statusDiv.removeAttribute('title');
            
            if (type === 'success') {
                statusDiv.textContent = message;
                statusDiv.className = 'small status-success';
            } else if (type === 'error') {
                statusDiv.textContent = message;
                statusDiv.className = 'small status-error';
            } else {
                statusDiv.textContent = message;
                statusDiv.className = 'small';
            }
            
            if (type !== 'error') {
                setTimeout(() => {
                    statusDiv.textContent = '';
                    statusDiv.className = 'small';
                }, 5000);
            }
        },
        
        showError: function(message) {
            this.showStatus(message, 'error');
        },
        
        setButtonLoading: function(button, loading) {
            if (typeof window.setButtonLoading === 'function') {
                window.setButtonLoading(button, loading);
            } else {
                if (!button) return;
                if (loading) {
                    button.disabled = true;
                    button.setAttribute('aria-busy', 'true');
                    button.classList.add('opacity-75');
                } else {
                    button.disabled = false;
                    button.removeAttribute('aria-busy');
                    button.classList.remove('opacity-75');
                }
            }
        }
    };
})();

