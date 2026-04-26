<?php

namespace App\Http\Controllers\HabitTracker;

use App\Actions\HabitTracker\ToggleTick;
use App\Http\Controllers\Controller;
use App\Http\Requests\HabitTracker\ToggleTickRequest;
use App\Models\Habit;
use Illuminate\Http\RedirectResponse;

class TickController extends Controller
{
    public function __construct(private ToggleTick $toggleTick) {}

    /**
     * Toggle the existence of a tick for one (habit, day).
     */
    public function toggle(ToggleTickRequest $request): RedirectResponse
    {
        $habit = Habit::query()->findOrFail($request->integer('habit_id'));

        $result = $this->toggleTick->execute($habit, $request->string('tick_date')->toString());

        $request->session()->flash('success', 'Tick updated');

        if ($result['just_achieved_reward'] !== null) {
            $habit->loadMissing('sprintParticipant.person');

            $reward = $result['just_achieved_reward'];
            $person = $habit->sprintParticipant->person;

            $request->session()->flash('achievement', [
                'person_id' => $person->id,
                'reward_id' => $reward->id,
                'person_name' => $person->name,
                'reward_name' => $reward->name,
            ]);
        }

        return back();
    }
}
