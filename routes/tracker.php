<?php

use App\Http\Controllers\HabitTracker\HabitController;
use App\Http\Controllers\HabitTracker\HistoryController;
use App\Http\Controllers\HabitTracker\PersonController;
use App\Http\Controllers\HabitTracker\RewardController;
use App\Http\Controllers\HabitTracker\SettingsController;
use App\Http\Controllers\HabitTracker\SprintController;
use App\Http\Controllers\HabitTracker\SprintParticipantController;
use App\Http\Controllers\HabitTracker\TickController;
use App\Http\Controllers\HabitTracker\TotalsController;
use App\Http\Controllers\HabitTracker\TrackerController;
use Illuminate\Support\Facades\Route;

Route::get('/', [TrackerController::class, 'show'])->name('tracker.show');

Route::post('ticks', [TickController::class, 'toggle'])->name('ticks.toggle');

Route::post('sprints', [SprintController::class, 'store'])->name('sprints.store');
Route::post('sprints/{sprint}/end', [SprintController::class, 'end'])->name('sprints.end');

Route::get('history', [HistoryController::class, 'index'])->name('history.index');
Route::get('history/{sprint}', [HistoryController::class, 'show'])->name('history.show');

Route::get('totals', [TotalsController::class, 'show'])->name('totals.show');

Route::get('settings', [SettingsController::class, 'index'])->name('settings.index');

Route::post('people', [PersonController::class, 'store'])->name('people.store');
Route::patch('people/{person}', [PersonController::class, 'update'])->name('people.update');
Route::delete('people/{person}', [PersonController::class, 'destroy'])->name('people.destroy');

Route::post('sprints/{sprint}/participants', [SprintParticipantController::class, 'store'])
    ->name('sprint-participants.store');

Route::post('sprint-participants/{sprintParticipant}/habits', [HabitController::class, 'store'])
    ->name('habits.store');
Route::patch('habits/{habit}', [HabitController::class, 'update'])->name('habits.update');
Route::delete('habits/{habit}', [HabitController::class, 'destroy'])->name('habits.destroy');

Route::post('people/{person}/rewards', [RewardController::class, 'store'])->name('rewards.store');
Route::patch('rewards/{reward}', [RewardController::class, 'update'])->name('rewards.update');
Route::delete('rewards/{reward}', [RewardController::class, 'destroy'])->name('rewards.destroy');

Route::post('sprint-participants/{sprintParticipant}/active-reward', [SprintParticipantController::class, 'updateReward'])
    ->name('sprint-participants.update-reward');
