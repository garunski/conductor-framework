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
            
            let alertClass = 'alert-info';
            if (type === 'success') {
                alertClass = 'alert-success';
            } else if (type === 'error') {
                alertClass = 'alert-danger';
            }
            
            const alert = document.createElement('div');
            alert.className = `alert ${alertClass} mb-0`;
            alert.textContent = message;
            
            statusDiv.textContent = '';
            statusDiv.appendChild(alert);
            
            if (type !== 'error') {
                setTimeout(() => {
                    statusDiv.textContent = '';
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

