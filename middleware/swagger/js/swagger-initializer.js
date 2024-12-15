window.onload = function() {
  //<editor-fold desc="Changeable Configuration Block">
  console.log('swagger ui v5.18.2')
  // the following lines will be replaced by docker/configurator, when it runs in a docker-container
  window.ui = SwaggerUIBundle({
    url: "openapi.yaml",
    dom_id: '#swagger-ui',
    deepLinking: true,
    docExpansion: 'none',
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl
    ],
    layout: "StandaloneLayout",
    syntaxHighlight: {
      activated: true,
      theme: 'monokai'
    },
    validatorUrl: 'none',
    tryItOutEnabled: true
  });

  //</editor-fold>
};
