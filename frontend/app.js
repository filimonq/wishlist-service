'use strict';

const $ = (selector, root = document) => root.querySelector(selector);
const state = { token: sessionStorage.getItem('wishlist-token') || '', lists: [], selected: null, items: [], publicToken: '', register: false, editWishlist: null, editItem: null, pending: 0 };
const errors = {
  'invalid credentials': 'Неверный email или пароль.',
  'user already exists': 'Этот email уже зарегистрирован.',
  'event date must be in the future': 'Выберите дату события в будущем.',
  'item already reserved': 'Этот подарок уже забронировали.',
  'wishlist not found': 'Вишлист не найден.',
  'internal server error': 'Ошибка сервера. Попробуйте ещё раз.'
};

function message(text, error = false) {
  const notice = $('#notice');
  notice.textContent = text;
  notice.hidden = !text;
  notice.classList.toggle('error', error);
  const field = $('dialog[open] .form-error');
  if (field) { field.textContent = error ? text : ''; field.hidden = !error; }
}

async function run(action) {
  if (state.pending) return;
  state.pending++;
  message('');
  const buttons = [...document.querySelectorAll('button')];
  const disabled = buttons.map(button => button.disabled);
  buttons.forEach(button => { button.disabled = true; });
  try { await action(); }
  catch (error) { message(error.message, true); }
  finally {
    buttons.forEach((button, index) => { button.disabled = disabled[index]; });
    state.pending--;
  }
}

async function api(path, method = 'GET', data, authenticated = true) {
  const headers = {};
  if (data !== undefined) headers['Content-Type'] = 'application/json';
  if (authenticated && state.token) headers.Authorization = `Bearer ${state.token}`;
  let response;
  try { response = await fetch(`/api/v1${path}`, { method, headers, body: data === undefined ? undefined : JSON.stringify(data) }); }
  catch { throw new Error('Не удалось связаться с сервером. Проверьте подключение.'); }
  const body = response.status === 204 ? null : await response.json();
  if (!response.ok) {
    if (response.status === 401 && authenticated) {
      logout();
      throw new Error('Сессия закончилась. Войдите снова.');
    }
    const fallback = response.status === 400 ? 'Проверьте заполненные поля.' : response.status === 409 ? 'Данные уже изменились или подарок забронирован. Обновите список.' : 'Не удалось выполнить действие.';
    throw new Error(errors[body?.error] || fallback);
  }
  return body;
}

function setView(name) {
  ['auth', 'owner', 'public'].forEach(view => { $(`#${view}-view`).hidden = view !== name; });
  $('#logout').hidden = !state.token;
}

function logout() {
  state.token = '';
  state.selected = null;
  state.lists = [];
  sessionStorage.removeItem('wishlist-token');
  document.querySelectorAll('dialog[open]').forEach(dialog => dialog.close());
  setView(state.publicToken ? 'public' : 'auth');
}

function localDate(value) {
  const date = new Date(value);
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
}

function displayDate(value) {
  return new Date(value).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' });
}

function shareLink(token) {
  const url = new URL('/', location.href);
  url.hash = `public/${token}`;
  return url.href;
}

async function loadLists(selectedID = state.selected?.id) {
  state.lists = await api('/wishlists');
  const root = $('#wishlist-list');
  root.replaceChildren();
  for (const list of state.lists) {
    const button = document.createElement('button');
    button.className = 'wishlist-button';
    button.dataset.id = list.id;
    button.append(document.createTextNode(list.title));
    const date = document.createElement('small');
    date.textContent = displayDate(list.event_date);
    button.append(date);
    button.onclick = () => run(() => selectWishlist(list.id));
    root.append(button);
  }
  if (!state.lists.length) {
    state.selected = null;
    root.textContent = 'Списков пока нет.';
    const empty = document.createElement('p');
    empty.className = 'empty';
    empty.textContent = 'Создай первый вишлист — здесь появятся твои подарки.';
    $('#wishlist-detail').replaceChildren(empty);
    return;
  }
  const selected = state.lists.find(list => list.id === selectedID) || state.lists[0];
  await selectWishlist(selected.id);
}

async function selectWishlist(id) {
  const [list, items] = await Promise.all([api(`/wishlists/${id}`), api(`/wishlists/${id}/items`)]);
  state.selected = list;
  state.items = items || [];
  document.querySelectorAll('.wishlist-button').forEach(button => { button.classList.toggle('selected', button.dataset.id === id); });
  renderDetail($('#wishlist-detail'), list, state.items, false);
}

async function loadPublic() {
  setView('public');
  const list = await api(`/public/${encodeURIComponent(state.publicToken)}`, 'GET', undefined, false);
  renderDetail($('#public-detail'), list, list.items || [], true);
}

function renderDetail(root, list, items, publicView) {
  const fragment = $('#detail-template').content.cloneNode(true);
  $('.detail-title', fragment).textContent = list.title;
  $('.detail-date', fragment).textContent = `Событие · ${displayDate(list.event_date)}`;
  $('.detail-description', fragment).textContent = list.description;
  const ownerActions = $('.owner-actions', fragment);
  ownerActions.hidden = publicView;
  if (!publicView) {
    const link = shareLink(list.public_token);
    $('.share-link', fragment).value = link;
    $('.open-public', fragment).href = link;
    $('.copy-link', fragment).onclick = () => run(async () => {
      try { await navigator.clipboard.writeText(link); message('Ссылка скопирована.'); }
      catch { $('.share-link', root).select(); message('Скопируйте выделенную ссылку: Ctrl+C или ⌘C.'); }
    });
    $('.add-item', fragment).onclick = () => openItemDialog();
    $('.edit-wishlist', fragment).onclick = () => openWishlistDialog(list);
    $('.delete-wishlist', fragment).onclick = () => run(async () => {
      if (!confirm('Удалить вишлист вместе со всеми подарками?')) return;
      await api(`/wishlists/${list.id}`, 'DELETE');
      await loadLists();
      message('Вишлист удалён.');
    });
  }
  const container = $('.items', fragment);
  if (!items.length) {
    const empty = document.createElement('p');
    empty.className = 'empty';
    empty.textContent = publicView ? 'В этом списке пока нет подарков.' : 'Добавь первый подарок в этот список.';
    container.append(empty);
  }
  for (const item of items) {
    const card = $('#item-template').content.cloneNode(true);
    $('.item-name', card).textContent = item.name;
    $('.item-description', card).textContent = item.description;
    $('.item-priority', card).textContent = `Приоритет: ${item.priority} из 5`;
    const badge = $('.badge', card);
    badge.textContent = item.is_reserved ? 'Забронирован' : 'Свободен';
    badge.classList.toggle('reserved', item.is_reserved);
    if (item.url) {
      try {
        const url = new URL(item.url);
        if (['http:', 'https:'].includes(url.protocol)) {
          $('.item-link', card).href = url.href;
          $('.item-link', card).hidden = false;
        }
      } catch { /* Некорректная ссылка не становится активной. */ }
    }
    const edit = $('.edit-item', card);
    const remove = $('.delete-item', card);
    const reserve = $('.reserve-item', card);
    edit.hidden = remove.hidden = publicView;
    reserve.hidden = !publicView;
    reserve.disabled = item.is_reserved;
    if (item.is_reserved) reserve.textContent = 'Уже забронирован';
    edit.onclick = () => openItemDialog(item);
    remove.onclick = () => run(async () => {
      if (!confirm('Удалить подарок из списка?')) return;
      await api(`/wishlists/${list.id}/items/${item.id}`, 'DELETE');
      await selectWishlist(list.id);
      message('Подарок удалён.');
    });
    reserve.onclick = () => run(async () => {
      if (!confirm('Забронировать этот подарок? Отменить бронирование через интерфейс нельзя.')) return;
      try { await api(`/public/${state.publicToken}/items/${item.id}/reserve`, 'POST', undefined, false); }
      catch (error) { await loadPublic(); throw error; }
      await loadPublic();
      message('Подарок забронирован за вами.');
    });
    container.append(card);
  }
  root.replaceChildren(fragment);
}

function openWishlistDialog(list = null) {
  state.editWishlist = list;
  const form = $('#wishlist-form');
  form.reset();
  form.elements.title.value = list?.title || '';
  form.elements.description.value = list?.description || '';
  form.elements.event_date.value = localDate(list?.event_date || Date.now() + 7 * 86400000);
  $('#wishlist-form-title').textContent = list ? 'Изменить вишлист' : 'Новый вишлист';
  $('.form-error', form).hidden = true;
  $('#wishlist-dialog').showModal();
}

function openItemDialog(item = null) {
  state.editItem = item;
  const form = $('#item-form');
  form.reset();
  for (const key of ['name', 'description', 'url']) form.elements[key].value = item?.[key] || '';
  form.elements.priority.value = item?.priority || 3;
  $('#item-form-title').textContent = item ? 'Изменить подарок' : 'Добавить подарок';
  $('.form-error', form).hidden = true;
  $('#item-dialog').showModal();
}

function authMode(register) {
  state.register = register;
  $('#login-tab').classList.toggle('active', !register);
  $('#register-tab').classList.toggle('active', register);
  $('#login-tab').setAttribute('aria-pressed', String(!register));
  $('#register-tab').setAttribute('aria-pressed', String(register));
  $('#auth-submit').textContent = register ? 'Зарегистрироваться' : 'Войти';
  $('#password').minLength = register ? 8 : 1;
  $('#password').autocomplete = register ? 'new-password' : 'current-password';
  message('');
}

$('#login-tab').onclick = () => authMode(false);
$('#register-tab').onclick = () => authMode(true);
$('#logout').onclick = () => { logout(); message('Вы вышли из аккаунта.'); };
$('#create-wishlist').onclick = () => openWishlistDialog();
document.querySelectorAll('[data-close]').forEach(button => { button.onclick = () => document.getElementById(button.dataset.close).close(); });

$('#auth-form').onsubmit = event => {
  event.preventDefault();
  run(async () => {
    const form = event.target;
    const credentials = { email: form.elements.email.value.trim(), password: form.elements.password.value };
    if (state.register) await api('/auth/register', 'POST', credentials, false);
    const result = await api('/auth/login', 'POST', credentials, false);
    state.token = result.token;
    sessionStorage.setItem('wishlist-token', state.token);
    form.reset();
    setView('owner');
    await loadLists();
  });
};

$('#wishlist-form').onsubmit = event => {
  event.preventDefault();
  run(async () => {
    const fields = event.target.elements;
    const title = fields.title.value.trim();
    const date = new Date(fields.event_date.value);
    if (!title) throw new Error('Введите название списка.');
    if (!(date.getTime() > Date.now())) throw new Error('Выберите дату события в будущем.');
    const data = { title, description: fields.description.value, event_date: date.toISOString() };
    const path = state.editWishlist ? `/wishlists/${state.editWishlist.id}` : '/wishlists';
    const list = await api(path, state.editWishlist ? 'PUT' : 'POST', data);
    $('#wishlist-dialog').close();
    await loadLists(list.id);
    message('Вишлист сохранён.');
  });
};

$('#item-form').onsubmit = event => {
  event.preventDefault();
  run(async () => {
    const fields = event.target.elements;
    const name = fields.name.value.trim();
    if (!name) throw new Error('Введите название подарка.');
    const data = { name, description: fields.description.value, url: fields.url.value.trim(), priority: Number(fields.priority.value) };
    const path = `/wishlists/${state.selected.id}/items${state.editItem ? `/${state.editItem.id}` : ''}`;
    await api(path, state.editItem ? 'PUT' : 'POST', data);
    $('#item-dialog').close();
    await selectWishlist(state.selected.id);
    message('Подарок сохранён.');
  });
};

async function navigate() {
  message('');
  const match = location.hash.match(/^#public\/([0-9a-f-]{36})$/i);
  state.publicToken = match ? match[1] : '';
  if (state.publicToken) {
    $('#public-detail').replaceChildren();
    await loadPublic();
  } else if (state.token) {
    setView('owner');
    await loadLists();
  } else { setView('auth'); }
}

window.addEventListener('hashchange', () => run(navigate));
run(navigate);
