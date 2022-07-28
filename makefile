migrate-up:
	export $(cat .env.local | xargs) &> /dev/null && sql-migrate up -config=dbconfig.yml -env=development

migrate-down:
	export $(cat .env.local | xargs) &> /dev/null && sql-migrate down -config=dbconfig.yml -env=development