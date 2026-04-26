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

class HistoryAndTotalsControllerTest extends TestCase
{
    use LazilyRefreshDatabase;

    public function test_history_index_lists_archived_sprints(): void
    {
        Sprint::factory()->archived()->create([
            'start_date' => '2026-01-01',
            'end_date' => '2026-01-28',
            'ended_at' => '2026-01-28 00:00:00',
        ]);
        Sprint::factory()->archived()->create([
            'start_date' => '2026-03-01',
            'end_date' => '2026-03-28',
            'ended_at' => '2026-03-28 00:00:00',
        ]);

        $this->get(route('history.index'))
            ->assertOk()
            ->assertInertia(fn (AssertableInertia $page) => $page
                ->component('history/Index')
                ->has('sprints', 2)
                ->where('sprints.0.start_date', '2026-03-01'),
            );
    }

    public function test_history_show_renders_archived_grid(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $person = Person::factory()->create();
        $sprint = Sprint::factory()->archived()->create([
            'start_date' => '2026-03-01',
            'end_date' => '2026-03-04',
            'ended_at' => '2026-03-04 00:00:00',
        ]);
        $participant = SprintParticipant::factory()->for($sprint)->for($person)->create();
        Habit::factory()->for($participant, 'sprintParticipant')->create(['name' => 'Read']);

        $this->get(route('history.show', $sprint))
            ->assertOk()
            ->assertInertia(fn (AssertableInertia $page) => $page
                ->component('history/Show')
                ->where('sprint.id', $sprint->id)
                ->where('sprint.status', Sprint::STATUS_ARCHIVED),
            );
    }

    public function test_history_show_404s_for_active_sprint(): void
    {
        $sprint = Sprint::factory()->create();

        $this->get(route('history.show', $sprint))
            ->assertNotFound();
    }

    public function test_totals_show_returns_per_person_lifetime_data(): void
    {
        $person = Person::factory()->create(['name' => 'Bob', 'display_order' => 1]);
        $sprint = Sprint::factory()->archived()->create();
        $participant = SprintParticipant::factory()->for($sprint)->for($person)->create();
        $habit = Habit::factory()->for($participant, 'sprintParticipant')->create();
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-03-01']);
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-03-02']);
        Reward::factory()->for($person)->achieved()->create();

        $this->get(route('totals.show'))
            ->assertOk()
            ->assertInertia(fn (AssertableInertia $page) => $page
                ->component('totals/Show')
                ->has('people', 1)
                ->where('people.0.total_ticks_earned', 2)
                ->where('people.0.total_rewards_achieved', 1)
                ->where('people.0.sprints_participated_count', 1),
            );
    }
}
