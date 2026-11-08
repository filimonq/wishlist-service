#!/usr/bin/env python3
"""Prepare the ignored local Alertmanager Secret without printing the bot token."""

import getpass
import json
import os
from pathlib import Path
import re
import tempfile
import urllib.error
import urllib.request


def main():
    token = getpass.getpass('Токен Telegram-бота (ввод скрыт): ').strip()
    if not re.fullmatch(r'[0-9]+:[A-Za-z0-9_-]+', token):
        raise SystemExit('Некорректный формат токена')
    try:
        with urllib.request.urlopen(
            'https://api.telegram.org/bot' + token + '/getUpdates', timeout=15
        ) as response:
            updates = json.load(response)
    except (urllib.error.URLError, TimeoutError):
        raise SystemExit('Не удалось получить сообщения бота. Проверьте токен и доступ к Telegram.') from None
    if not updates.get('ok'):
        raise SystemExit('Telegram не вернул сообщения бота')
    chats = {
        update['message']['chat']['id']
        for update in updates.get('result', [])
        if update.get('message', {}).get('chat', {}).get('type') == 'private'
    }
    if not chats:
        raise SystemExit('Откройте своего бота, отправьте /start и повторите команду.')
    if len(chats) == 1:
        chat_id = chats.pop()
    else:
        print('Найдены личные чаты:', ', '.join(str(chat) for chat in sorted(chats)))
        try:
            chat_id = int(input('ID вашего чата: ').strip())
        except ValueError:
            raise SystemExit('ID должен быть целым числом') from None
        if chat_id not in chats:
            raise SystemExit('Этот чат отсутствует в сообщениях бота')
    directory = Path(__file__).resolve().parent
    content = (directory / 'alertmanager.example.yaml').read_text()
    content = content.replace('CHANGE_ME_BOT_TOKEN', token).replace('CHANGE_ME_CHAT_ID', str(chat_id))
    with tempfile.NamedTemporaryFile(mode='w', encoding='utf-8', dir=directory, prefix='.alertmanager-', delete=False) as output:
        output.write(content)
        staging = Path(output.name)
    try:
        staging.chmod(0o600)
        os.replace(staging, directory / 'alertmanager.local.yaml')
    finally:
        staging.unlink(missing_ok=True)
    print('Создан k8s/alerting/alertmanager.local.yaml. Токен не выводится и исключён из Git.')


if __name__ == '__main__':
    main()
