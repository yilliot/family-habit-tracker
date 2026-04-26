<?php

use Illuminate\Support\Facades\Route;

Route::middleware(['auth', 'verified'])->group(function () {
    Route::inertia('dashboard', 'dashboard')->name('dashboard');
});

require __DIR__.'/settings.php';

// Load Habit Tracker routes last so that the public `settings.index` route
// takes precedence over the auth-scoped `Route::redirect('settings', ...)`
// scaffolding inside settings.php (Laravel's RouteCollection keys by URI/method,
// so the most-recently-registered match wins).
require __DIR__.'/tracker.php';
