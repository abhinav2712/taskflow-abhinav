FROM nginx:alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
RUN printf '%s\n' \
  '<!doctype html>' \
  '<html lang="en">' \
  '  <head>' \
  '    <meta charset="utf-8" />' \
  '    <meta name="viewport" content="width=device-width, initial-scale=1.0" />' \
  '    <title>TaskFlow</title>' \
  '  </head>' \
  '  <body>' \
  '    <main>' \
  '      <h1>TaskFlow</h1>' \
  '      <p>Frontend container placeholder for the initial scaffold.</p>' \
  '    </main>' \
  '  </body>' \
  '</html>' \
  > /usr/share/nginx/html/index.html
