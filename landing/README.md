# Deploy the landing page

This is a buildless static site. Deploy the contents of this directory, keeping index.html and assets/ at the web root. No Node.js runtime, installation, environment secrets or app backend is required.

## Static hosting

Point your host's publish directory to landing. Leave the build command empty. This works with a static web server, GitHub Pages, Cloudflare Pages, Netlify or similar hosting. The relative asset paths support a domain root or a project subpath.

The GitHub Pages workflow is manual: select Deploy landing page in Actions and run it after enabling Pages with GitHub Actions as its source. CI validates the page on normal code pushes without deploying it.

## Domain: blacksand.team

The canonical URL, sharing metadata, sitemap and CNAME file are prepared for blacksand.team. Add that domain in your hosting provider's settings, follow its DNS instructions and enable HTTPS. The project does not configure or verify your DNS.

For GitHub Pages, enter blacksand.team in repository Settings → Pages and apply the DNS records GitHub specifies. The CNAME file alone does not activate a custom domain for a GitHub Actions deployment.

## Preview and validation

Run node tools/preview-landing.mjs from the repository root and open http://127.0.0.1:4173. Alternatively serve this folder with any static HTTP server. From the repository root, run node tools/validate-landing.mjs and node --check landing/assets/js/app.js. Clipboard access needs a secure context or localhost; a manual selection fallback is available.
