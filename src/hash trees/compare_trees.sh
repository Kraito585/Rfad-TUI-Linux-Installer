#!/usr/bin/env bash

BASE="base_tree.txt"
ENB="enb_tree.txt"
RESHADE="reshade_tree.txt"

if [[ ! -f "$BASE" ]]; then
    echo "Ошибка: Базовый файл $BASE не найден. Сравнивать не с чем."
    exit 1
fi

# Функция умного сравнения через awk
compare() {
    local base_txt=$1
    local mod_txt=$2
    local name=$3

    if [[ ! -f "$mod_txt" ]]; then
        echo "Файл $mod_txt не найден, пропускаем $name."
        return
    fi

    local out_log="diff_${name}.txt"
    local out_raw="diff_${name}_raw.txt"

    echo "==> Сравнение $base_txt и $mod_txt ($name)..."

    # Удаляем старые отчеты, если они были
    rm -f "$out_log" "$out_raw"

    awk -v logfile="$out_log" -v rawfile="$out_raw" '
    # Читаем первый файл (base_tree.txt) и запоминаем все пути и их хэши
    NR==FNR {
        hash = $NF
        path = $0
        sub(/ [a-f0-9]+$/, "", path) # Аккуратно отрезаем хэш, сохраняя пробелы в пути
        base[path] = hash
        next
    }
    # Читаем второй файл (модифицированную версию)
    {
        hash = $NF
        path = $0
        sub(/ [a-f0-9]+$/, "", path)
        mod[path] = 1

        if (!(path in base)) {
            print "[+] ДОБАВЛЕН: " path > logfile
            print path " " hash > rawfile
        } else if (base[path] != hash) {
            print "[~] ИЗМЕНЕН:  " path " (Хэш изменился)" > logfile
            print path " " hash > rawfile
        }
    }
    # Проверяем, не удалил ли мод какие-то базовые файлы
    END {
        for (p in base) {
            if (!(p in mod)) {
                print "[-] УДАЛЕН:   " p > logfile
            }
        }
    }' "$base_txt" "$mod_txt"

    local added=$(grep -c "^\[+\]" "$out_log" 2>/dev/null || echo 0)
    local changed=$(grep -c "^\[~\]" "$out_log" 2>/dev/null || echo 0)
    local deleted=$(grep -c "^\[-\]" "$out_log" 2>/dev/null || echo 0)

    echo "    Анализ $name завершен:"
    echo "    Добавлено: $added | Изменено: $changed | Удалено: $deleted"
    echo "    -> Отчет для чтения: $out_log"
    echo "    -> Список для архиватора: $out_raw"
    echo "------------------------------------------------"
}

compare "$BASE" "$ENB" "enb"
compare "$BASE" "$RESHADE" "reshade"

echo "==> Все сравнения завершены!"
