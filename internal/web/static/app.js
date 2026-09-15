/*
 * Progressive enhancement for the family ledger.
 *
 * Everything on the page works without this file: each tick cell is a real
 * submit button inside a real form, and every settings form is a plain POST.
 * This script makes ticking feel instant on the tablet and replaces full page
 * reloads with targeted updates.
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
     * The cell flips the moment it is tapped, before the request is sent, so
     * the tablet never feels laggy. The server's response then replaces the
     * cell with the authoritative state; if the request fails the flip is
     * rolled back and the reason is shown.
     */
    function flip(button) {
        var next = button.dataset.ticked !== 'true';
        button.dataset.ticked = next ? 'true' : 'false';
        button.setAttribute('aria-pressed', next ? 'true' : 'false');
        button.dataset.pending = 'true';

        return next;
    }

    function rollback(button, previous) {
        button.dataset.ticked = previous ? 'true' : 'false';
        button.setAttribute('aria-pressed', previous ? 'true' : 'false');
        delete button.dataset.pending;
    }

    // htmx reports the form as the requesting element, not the button inside it.
    function cellButton(event) {
        var elt = event.detail.elt;

        return elt && elt.querySelector ? elt.querySelector('.ht-cell-btn') : null;
    }

    document.body.addEventListener('htmx:beforeRequest', function (event) {
        var button = cellButton(event);

        if (button) {
            button.dataset.wasTicked = button.dataset.ticked === 'true' ? 'true' : 'false';
            flip(button);
        }
    });

    function failed(event) {
        var button = cellButton(event);

        if (button) {
            rollback(button, button.dataset.wasTicked === 'true');
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
    // missed and the dialog can be dismissed from the keyboard.
    document.body.addEventListener('htmx:afterSwap', function (event) {
        if (event.detail.target && event.detail.target.id === 'celebration') {
            var modal = event.detail.target.querySelector('.ht-modal');

            if (modal && !modal.hidden) {
                var targets = focusables(modal);
                if (targets.length) {
                    targets[0].focus();
                }
            }
        }
    });

    /* ——— Sprint dates ——————————————————————————————————————————— */

    /*
     * The start-sprint form is the only one carrying freshly typed text (each
     * person's habits), so a round trip to the server just to be told the end
     * date precedes the start would throw that work away. Bind the two inputs
     * together instead; the server still enforces the same rule.
     */
    document.addEventListener('DOMContentLoaded', function () {
        var start = document.querySelector('input[name="start_date"]');
        var end = document.querySelector('input[name="end_date"]');

        if (!start || !end) {
            return;
        }

        function sync() {
            end.min = start.value;

            if (end.value && end.value < start.value) {
                end.value = start.value;
            }
        }

        start.addEventListener('change', sync);
        sync();
    });

    /* ——— Grid convenience ——————————————————————————————————————— */

    // Scroll today's column into view on a long sprint, so the tablet opens on
    // the day someone actually wants to tap.
    document.addEventListener('DOMContentLoaded', function () {
        var today = document.querySelector('[data-today-col="true"]');
        var scroller = today && today.closest('.ht-scroll-x');

        if (!today || !scroller || scroller.scrollWidth <= scroller.clientWidth) {
            return;
        }

        var offset = today.offsetLeft - scroller.clientWidth / 2 + today.offsetWidth / 2;
        scroller.scrollLeft = Math.max(0, offset);
    });
})();
