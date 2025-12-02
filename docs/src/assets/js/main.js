import * as bootstrap from 'bootstrap'

// Fetch GitHub star count
async function fetchGitHubStars() {
    const starCountEl = document.querySelector('.github-star-count');
    if (!starCountEl) return;
    
    const repo = starCountEl.dataset.repo;
    try {
        const response = await fetch(`https://api.github.com/repos/${repo}`);
        if (response.ok) {
            const data = await response.json();
            starCountEl.textContent = data.stargazers_count;
            starCountEl.style.display = 'inline';
        }
    } catch (error) {
        // Silently fail - just don't show the count
        console.debug('Failed to fetch GitHub stars:', error);
    }
}

// Fetch stars when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', fetchGitHubStars);
} else {
    fetchGitHubStars();
}