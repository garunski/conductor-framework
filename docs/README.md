# Conductor Framework Documentation

This directory contains the documentation site for Conductor Framework, built with Vituum, PostHTML, and Bootstrap.

## Setup

1. Install dependencies:
```bash
npm install
```

2. Start development server:
```bash
npm run dev
```

3. Build for production:
```bash
npm run build
```

4. Preview production build:
```bash
npm run preview
```

## Project Structure

```
docs/
├── src/
│   ├── pages/          # HTML pages
│   ├── layouts/        # Layout templates
│   ├── components/     # Reusable components
│   └── assets/         # CSS and JS files
├── vite.config.js      # Vite configuration
└── package.json        # Dependencies
```

## Pages

- `index.html` - Homepage
- `examples.html` - Example explanations
- `agents.html` - AI agent prompts examples
- `contributing.html` - Contribution guidelines
- `implementation.html` - Implementation details
- `design.html` - Design concepts

## Deployment

The built site is output to the `dist/` directory and can be deployed to GitHub Pages or any static hosting service.

