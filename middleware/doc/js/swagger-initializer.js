window.onload = function () {
  //<editor-fold desc="Changeable Configuration Block">
  let href = window.location.origin + window.location.pathname;
  if (href[href.length - 1] != "/") {
    href += "/"
  }
  console.log(href)
  // the following lines will be replaced by docker/configurator, when it runs in a docker-container
  window.ui = SwaggerUIBundle({
    urls: [{ url: "openapi.yaml", name: "api" }],
    dom_id: '#swagger-ui',
    docExpansion: false,
    defaultModelsExpandDepth: -1,
    validatorUrl: null,
    deepLinking: true,
    filter: true,
    showCommonExtensions: true,
    tryItOutEnabled: true,
    displayRequestDuration: true,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      //SwaggerUIBundle.plugins.DownloadUrl
    ],
    layout: "StandaloneLayout"
  });

  //</editor-fold>
};
