package swagger

// DefaultInitializer is the default swagger-initializer.js content
// that loads specs dynamically from the /openapi/resources endpoint.
const DefaultInitializer = `window.onload = function () {
  const getUI = (baseUrl, resources) => {
    var swaggerOptions = {
      dom_id: '#swagger-ui',
      url: baseUrl + '/openapi/specs',
      urls: resources && resources.length > 0 ? resources : undefined,
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIStandalonePreset
      ],
      plugins: [
        SwaggerUIBundle.plugins.DownloadUrl,
      ],
      layout: "StandaloneLayout",
      docExpansion: "none",
      deepLinking: true,
      tagsSorter: function(a, b) {
        if (a === "Authentication") return -1;
        if (b === "Authentication") return 1;
        return a.localeCompare(b);
      },
      operationsSorter: "alpha",
    };

    return SwaggerUIBundle(swaggerOptions);
  };

  const buildSystemAsync = async baseUrl => {
    try {
      var request;

      request = await fetch('/openapi/resources', {
        credentials: 'same-origin',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
        },
      });

      const resources = await request.json();
      window.ui = getUI(baseUrl, resources);
    } catch (err) {
      console.error('Error loading Swagger UI: ', err);
    }
  };

  const getBaseURL = () => {
    var url = window.location.search.match(/url=([^&]+)/);
    if (url && url.length > 1) {
      url = decodeURIComponent(url[1]);
    } else {
      url = window.location.origin;
    }
    return url;
  };

  (async () => {
    await buildSystemAsync(getBaseURL());
  })();
};
`

// DefaultCSS is the default custom CSS that hides the swagger branding.
const DefaultCSS = `.swagger-ui .info .main > a, .swagger-ui .info .title > span {
    display: none;
}
`

// SimpleInitializer is a simpler initializer for single-spec usage.
const SimpleInitializer = `window.onload = function() {
  window.ui = SwaggerUIBundle({
    url: "/openapi/specs",
    dom_id: '#swagger-ui',
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl
    ],
    layout: "StandaloneLayout",
    docExpansion: "none",
    deepLinking: true,
    operationsSorter: "alpha",
    tagsSorter: "alpha"
  });
};
`
