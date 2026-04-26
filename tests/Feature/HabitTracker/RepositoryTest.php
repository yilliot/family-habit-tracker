<?php

namespace Tests\Feature\HabitTracker;

use App\Models\Habit;
use App\Models\Person;
use App\Models\Reward;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use App\Models\Tick;
use App\Repositories\HabitTracker\HabitTrackerRepository;
use Illuminate\Foundation\Testing\LazilyRefreshDatabase;
use Illuminate\Support\Carbon;
use Tests\TestCase;

class RepositoryTest extends TestCase
{
    use LazilyRefreshDatabase;

    public function test_get_active_sprint_returns_null_when_none(): void
    {
        $repo = app(HabitTrackerRepository::class);

        $this->assertNull($repo->getActiveSprint());
    }

    public function test_get_active_sprint_grid_returns_full_shape(): void
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

        $grid = app(HabitTrackerRepository::class)->getActiveSprintGrid();

        $this->assertNotNull($grid);
        $this->assertSame($sprint->id, $grid['sprint']['id']);
        $this->assertCount(4, $grid['days']); // Apr 19,20,21,22
        $this->assertCount(1, $grid['participants']);
        $this->assertSame(1, $grid['participants'][0]['points_balance']);
        $this->assertSame('Read', $grid['participants'][0]['habits'][0]['name']);
        $this->assertSame(['2026-04-20'], $grid['participants'][0]['habits'][0]['tick_dates']);
        $this->assertNotNull($grid['participants'][0]['active_reward']);
    }

    public function test_get_person_balance_lifetime(): void
    {
        $person = Person::factory()->create();
        $sprint = Sprint::factory()->archived()->create();
        $participant = SprintParticipant::factory()->for($sprint)->for($person)->create();
        $habit = Habit::factory()->for($participant, 'sprintParticipant')->create();
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-03-01']);
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-03-02']);

        Reward::factory()->for($person)->achieved()->create(['cost' => 1]);

        $balance = app(HabitTrackerRepository::class)->getPersonBalance($person);

        $this->assertSame(1, $balance);
    }

    public function test_list_archived_sprints_orders_most_recent_first(): void
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

        $rows = app(HabitTrackerRepository::class)->listArchivedSprints();

        $this->assertSame('2026-03-01', $rows[0]['start_date']);
        $this->assertSame('2026-01-01', $rows[1]['start_date']);
    }

    public function test_lifetime_totals_per_person(): void
    {
        $person = Person::factory()->create(['display_order' => 1]);
        $sprint = Sprint::factory()->archived()->create();
        $participant = SprintParticipant::factory()->for($sprint)->for($person)->create();
        $habit = Habit::factory()->for($participant, 'sprintParticipant')->create();
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-03-01']);
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-03-02']);
        Reward::factory()->for($person)->achieved()->create();

        $rows = app(HabitTrackerRepository::class)->getLifetimeTotalsPerPerson();

        $this->assertSame(2, $rows[0]['total_ticks_earned']);
        $this->assertSame(1, $rows[0]['total_rewards_achieved']);
        $this->assertSame(1, $rows[0]['sprints_participated_count']);
    }

    public function test_list_people_excludes_trashed_and_orders(): void
    {
        Person::factory()->create(['name' => 'B', 'display_order' => 2]);
        Person::factory()->create(['name' => 'A', 'display_order' => 1]);
        Person::factory()->create(['name' => 'C', 'display_order' => 99])->delete();

        $people = app(HabitTrackerRepository::class)->listPeople();

        $this->assertSame(['A', 'B'], $people->pluck('name')->all());
    }

    public function test_list_person_rewards_orders_active_first_then_achieved_desc(): void
    {
        $person = Person::factory()->create();
        $oldAchieved = Reward::factory()->for($person)->create([
            'name' => 'Old',
            'achieved_at' => Carbon::parse('2026-01-01'),
        ]);
        $newerAchieved = Reward::factory()->for($person)->create([
            'name' => 'Newer',
            'achieved_at' => Carbon::parse('2026-03-01'),
        ]);
        $active = Reward::factory()->for($person)->create(['name' => 'Active']);

        $rewards = app(HabitTrackerRepository::class)->listPersonRewards($person);

        $this->assertSame(['Active', 'Newer', 'Old'], $rewards->pluck('name')->all());
    }
}
