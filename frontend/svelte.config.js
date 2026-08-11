import adapter from '@sveltejs/adapter-static';

export default {
	kit: {
		adapter: adapter({
			pages: '../internal/server/frontend/build',
			assets: '../internal/server/frontend/build',
			fallback: 'index.html',
			precompress: false
		}),
		paths: {
			relative: false
		}
	}
};
