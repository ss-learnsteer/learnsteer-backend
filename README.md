# Database (From your Neon Connection String)
heroku config:set DB_HOST=ep-royal-resonance-a16aw770-pooler.ap-southeast-1.aws.neon.tech

heroku config:set DB_USER=neondb_owner

heroku config:set DB_PASSWORD=npg_M3PsE4XbTFrd

heroku config:set DB_NAME=neondb

heroku config:set DB_PORT=5432

# Security
heroku config:set JWT_SECRET=cw3USQN/QXovCxKL9ZexqxvKQK+Csxgb39ggL2Ym3vI=

# Go Specifics
heroku config:set GO_ENV=production

heroku config:set GO_INSTALL_PACKAGE_SPEC=./cmd/server