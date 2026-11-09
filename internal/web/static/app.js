/*
 * Progressive enhancement for the family ledger.
 *
 * Everything on the page works without this file: each habit's box is a real
 * submit button inside a real form, and every other form is a plain POST.
 * This script makes ticking feel instant and shows toasts and dialogs.
 */
(function () {
    'use strict';

    /* ——— Toasts ———————————————————————————————————————————————— */

    function toastHost() {
        var host = document.getElementById('ht-toasts');

        if (!host) {
            host = document.createElement('div');
            host.id = 'ht-toasts';
            host.className = 'ht-toasts';
            host.setAttribute('aria-live', 'polite');
            document.body.appendChild(host);
        }

        return host;
    }

    function toast(message, type) {
        if (!message) {
            return;
        }

        var el = document.createElement('div');
        el.className = 'ht-toast';
        el.dataset.type = type || 'success';
        el.setAttribute('role', type === 'error' ? 'alert' : 'status');
        el.textContent = message;
        toastHost().appendChild(el);

        setTimeout(function () {
            el.dataset.leaving = 'true';
            setTimeout(function () {
                el.remove();
            }, 200);
        }, type === 'error' ? 5000 : 2600);
    }

    // Flash messages rendered by the server on a full page load.
    document.addEventListener('DOMContentLoaded', function () {
        document.querySelectorAll('[data-flash]').forEach(function (node) {
            toast(node.dataset.flash, node.dataset.flashType);
            node.remove();
        });
    });

    // Flash messages pushed by htmx responses via the HX-Trigger header.
    document.body.addEventListener('ht:toast', function (event) {
        var detail = event.detail || {};
        toast(detail.message, detail.type);
    });

    /* ——— Optimistic ticking ————————————————————————————————————— */

    /*
     * A habit flips the moment it is tapped, before the request is sent, so
     * the tablet never feels laggy. The server then answers with the row and
     * the totals freshly rendered; if the request fails the flip is rolled
     * back and the reason is shown.
     */

    // htmx reports the form as the requesting element; the row is its parent.
    function habitRow(event) {
        var elt = event.detail.elt;

        return elt && elt.closest ? elt.closest('.ht-habit') : null;
    }

    function setDone(row, done) {
        var button = row.querySelector('.ht-check');

        row.dataset.done = done ? 'true' : 'false';
        if (button) {
            button.setAttribute('aria-pressed', done ? 'true' : 'false');
        }
    }

    document.body.addEventListener('htmx:beforeRequest', function (event) {
        var row = habitRow(event);

        if (row) {
            row.dataset.wasDone = row.dataset.done;
            row.dataset.pending = 'true';
            setDone(row, row.dataset.done !== 'true');
        }
    });

    function failed(event) {
        var row = habitRow(event);

        if (row) {
            setDone(row, row.dataset.wasDone === 'true');
            delete row.dataset.pending;
        }
    }

    document.body.addEventListener('htmx:responseError', failed);
    document.body.addEventListener('htmx:sendError', function (event) {
        failed(event);
        toast('Could not reach the server. Your tick was not saved.', 'error');
    });

    /* ——— Modals ———————————————————————————————————————————————— */

    /*
     * Modals are plain markup toggled with the hidden attribute. Opening one
     * remembers what had focus, moves focus inside, traps Tab within the
     * dialog, and restores focus on close.
     */
    var lastFocused = null;

    function focusables(modal) {
        return Array.prototype.filter.call(
            modal.querySelectorAll('a[href], button:not([disabled]), input, select, textarea'),
            function (el) {
                return el.offsetParent !== null;
            }
        );
    }

    function openModal(id) {
        var modal = document.getElementById(id);

        if (!modal) {
            return;
        }

        lastFocused = document.activeElement;
        modal.hidden = false;

        var targets = focusables(modal);
        if (targets.length) {
            targets[0].focus();
        }
    }

    function closeModal(modal) {
        if (!modal || modal.hidden) {
            return;
        }

        modal.hidden = true;

        if (lastFocused && typeof lastFocused.focus === 'function') {
            lastFocused.focus();
            lastFocused = null;
        }
    }

    document.addEventListener('click', function (event) {
        var opener = event.target.closest('[data-modal-open]');
        if (opener) {
            event.preventDefault();
            openModal(opener.dataset.modalOpen);

            return;
        }

        var closer = event.target.closest('[data-modal-close]');
        if (closer) {
            event.preventDefault();
            closeModal(closer.closest('.ht-modal'));

            return;
        }

        // A click on the backdrop itself dismisses the dialog.
        if (event.target.classList && event.target.classList.contains('ht-modal')) {
            closeModal(event.target);
        }
    });

    document.addEventListener('keydown', function (event) {
        var modal = document.querySelector('.ht-modal:not([hidden])');

        if (!modal) {
            return;
        }

        if (event.key === 'Escape') {
            closeModal(modal);

            return;
        }

        if (event.key !== 'Tab') {
            return;
        }

        var targets = focusables(modal);
        if (!targets.length) {
            return;
        }

        var first = targets[0];
        var last = targets[targets.length - 1];

        if (event.shiftKey && document.activeElement === first) {
            event.preventDefault();
            last.focus();
        } else if (!event.shiftKey && document.activeElement === last) {
            event.preventDefault();
            first.focus();
        }
    });

    // A celebration arriving mid-session should take focus, so the news is not
    // missed and the dialog can be dismissed from the keyboard. It arrives as
    // an out-of-band swap alongside the day.
    // The event names the element that was swapped out, so read the slot
    // afresh rather than through it.
    document.body.addEventListener('htmx:oobAfterSwap', function () {
        var modal = document.querySelector('#celebration .ht-modal:not([hidden])');

        if (modal) {
            var targets = focusables(modal);
            if (targets.length) {
                targets[0].focus();
            }
        }
    });
})();
