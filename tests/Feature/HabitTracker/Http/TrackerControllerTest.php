<?php

namespace Tests\Feature\HabitTracker\Http;

use App\Models\Habit;
use App\Models\Person;
use App\Models\Reward;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use App\Models\Tick;
use Illuminate\Foundation\Testing\LazilyRefreshDatabase;
use Illuminate\Support\Carbon;
use Inertia\Testing\AssertableInertia;
use Tests\TestCase;

class TrackerControllerTest extends TestCase
{
    use LazilyRefreshDatabase;

    public function test_show_renders_empty_state_when_no_active_sprint(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $this->get(route('tracker.show'))
            ->assertOk()
            ->assertInertia(fn (AssertableInertia $page) => $page
                ->component('tracker/Show')
                ->where('sprint', null)
                ->where('participants', [])
                ->where('today', '2026-04-26'),
            );
    }

    public function test_show_renders_active_sprint_grid(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $person = Person::factory()->create(['name' => 'Bob', 'display_order' => 1]);
        $sprint = Sprint::factory()->create([
            'start_date' => '2026-04-19',
            'end_date' => '2026-04-22',
            'started_at' => '2026-04-19 00:00:00',
        ]);
        $reward = Reward::factory()->for($person)->create(['name' => 'X', 'cost' => 100]);
        $participant = SprintParticipant::factory()
            ->for($sprint)
            ->for($person)
            ->create(['active_reward_id' => $reward->id]);
        $habit = Habit::factory()->for($participant, 'sprintParticipant')->create(['name' => 'Read']);
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-04-20']);

        $this->get(route('tracker.show'))
            ->assertOk()
            ->assertInertia(fn (AssertableInertia $page) => $page
                ->component('tracker/Show')
                ->where('sprint.id', $sprint->id)
                ->has('participants', 1)
                ->where('participants.0.points_balance', 1)
                ->where('participants.0.habits.0.name', 'Read'),
            );
    }
}
