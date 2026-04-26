<?php

namespace Tests\Feature\HabitTracker;

use App\Models\Person;
use App\Models\Sprint;
use Database\Seeders\FamilySeeder;
use Illuminate\Foundation\Testing\LazilyRefreshDatabase;
use Tests\TestCase;

class FamilySeederTest extends TestCase
{
    use LazilyRefreshDatabase;

    public function test_seeds_four_family_members_in_order(): void
    {
        $this->seed(FamilySeeder::class);

        $names = Person::query()->orderBy('display_order')->pluck('name')->all();

        $this->assertSame(['爸爸', '妈妈', '可遇', '奇乐'], $names);
        $this->assertSame(0, Sprint::count());
    }

    public function test_running_seeder_twice_is_idempotent(): void
    {
        $this->seed(FamilySeeder::class);
        $this->seed(FamilySeeder::class);

        $this->assertSame(4, Person::count());
    }
}
