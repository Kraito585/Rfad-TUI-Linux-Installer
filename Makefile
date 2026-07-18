.PHONY: build-gdrive build-cdn run-gdrive run-cdn clean

# --- СБОРКА БИНАРНИКОВ ---
build-gdrive:
	@echo "Сборка версии Google Drive..."
	go build -tags gdrive -o rfad-installer-gdrive .

build-cdn:
	@echo "Сборка версии CDN (S3)..."
	go build -tags cdn -o rfad-installer-cdn .

# --- БЫСТРЫЙ ЗАПУСК ДЛЯ ТЕСТОВ ---
run-gdrive:
	@echo "Запуск версии Google Drive..."
	go run -tags gdrive .

run-cdn:
	@echo "Запуск версии CDN (S3)..."
	go run -tags cdn .

# --- ОЧИСТКА ---
clean:
	rm -f rfad-installer-gdrive rfad-installer-cdn